package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"unicode/utf8"

	"golang.org/x/sys/windows"
)

const (
	maxPlayers     = 64
	boneStride     = 32
	maxBoneIndex   = 28
	boneBufferSize = (maxBoneIndex + 1) * boneStride
	lifeStateAlive = 0
)

var boneIndices = map[string]int{
	"head": 7, "neck_0": 6, "spine_1": 8, "spine_2": 3, "pelvis": 1,
	"arm_upper_L": 9, "arm_lower_L": 10, "hand_L": 11,
	"arm_upper_R": 13, "arm_lower_R": 14, "hand_R": 15,
	"leg_upper_L": 17, "leg_lower_L": 18, "ankle_L": 19,
	"leg_upper_R": 20, "leg_lower_R": 21, "ankle_R": 22,
}

type Matrix [4][4]float32

type Vector3 struct{ X, Y, Z float32 }

func (v Vector3) Dist(other Vector3) float32 {
	dx := v.X - other.X
	dy := v.Y - other.Y
	dz := v.Z - other.Z
	return float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
}

type Vector2 struct{ X, Y float32 }

type Rectangle struct{ Top, Left, Right, Bottom float32 }

type Entity struct {
	PawnAddr uintptr
	Health   int32
	Team     int32
	Name     string
	Position Vector2
	Bones    map[string]Vector2
	HeadPos  Vector3
	Distance float32
	Rect     Rectangle
}

type screenProjector struct {
	width, height float32
	view          [16]float32
}

func newScreenProjector(width, height float32, view [16]float32) screenProjector {
	return screenProjector{width: width, height: height, view: view}
}

func (sp screenProjector) worldToScreen(position Vector3) (float32, float32, bool) {
	if sp.width <= 0 || sp.height <= 0 {
		return 0, 0, false
	}
	m := sp.view
	w := m[12]*position.X + m[13]*position.Y + m[14]*position.Z + m[15]
	if w < 0.001 {
		return 0, 0, false
	}
	inv := 1.0 / w
	clipX := (m[0]*position.X + m[1]*position.Y + m[2]*position.Z + m[3]) * inv
	clipY := (m[4]*position.X + m[5]*position.Y + m[6]*position.Z + m[7]) * inv

	halfW := sp.width * 0.5
	halfH := sp.height * 0.5
	x := halfW + halfW*clipX
	y := halfH - halfH*clipY
	return x, y, true
}

func readPlayerName(procHandle windows.Handle, controller uintptr, offsets Offset) string {
	nameBytes, err := readAt(procHandle, controller+offsets.M_iszPlayerName, 128)
	if err != nil {
		return "Player"
	}
	end := 0
	for end < len(nameBytes) && nameBytes[end] != 0 {
		end++
	}
	if end == 0 || !utf8.Valid(nameBytes[:end]) {
		return "Player"
	}
	return string(nameBytes[:end])
}

func readPawnHandle(procHandle windows.Handle, controller uintptr, offsets Offset) (uint32, error) {
	var handle uint32
	if readSafe(procHandle, controller+offsets.M_hPawn, &handle) == nil && handle != 0 {
		return handle, nil
	}
	if readSafe(procHandle, controller+offsets.M_hPlayerPawn, &handle) != nil || handle == 0 {
		return 0, fmt.Errorf("no pawn handle")
	}
	return handle, nil
}

func readBones(procHandle windows.Handle, boneArray uintptr, sp screenProjector, allBones bool) (map[string]Vector2, Vector3, bool) {
	boneData, err := readAt(procHandle, boneArray, boneBufferSize)
	if err != nil || len(boneData) < boneBufferSize {
		return nil, Vector3{}, false
	}

	bones := make(map[string]Vector2)
	var headPos Vector3
	hasHead := false

	for boneName, boneIndex := range boneIndices {
		if !allBones && boneName != "head" && boneName != AimbotTarget {
			continue
		}
		offset := boneIndex * boneStride
		pos := Vector3{
			X: math.Float32frombits(binary.LittleEndian.Uint32(boneData[offset:])),
			Y: math.Float32frombits(binary.LittleEndian.Uint32(boneData[offset+4:])),
			Z: math.Float32frombits(binary.LittleEndian.Uint32(boneData[offset+8:])),
		}
		if pos.X == 0 && pos.Y == 0 && pos.Z == 0 {
			continue
		}
		if boneName == "head" {
			headPos = pos
			hasHead = true
		}
		if x, y, ok := sp.worldToScreen(pos); ok {
			bones[boneName] = Vector2{X: x, Y: y}
		}
	}

	return bones, headPos, hasHead
}

func getEntitiesInfo(procHandle windows.Handle, clientDll uintptr, sp screenProjector, offsets Offset) []Entity {
	entities := make([]Entity, 0, 16)

	entityList, err := readPtr(procHandle, clientDll+offsets.DwEntityList)
	if err != nil || entityList == 0 {
		return entities
	}
	localPlayerP, err := readPtr(procHandle, clientDll+offsets.DwLocalPlayerPawn)
	if err != nil || localPlayerP == 0 {
		return entities
	}

	var localPlayerSceneOrigin Vector3
	if readSafe(procHandle, localPlayerP+offsets.M_vOldOrigin, &localPlayerSceneOrigin) != nil {
		return entities
	}

	var localTeam int32
	if readSafe(procHandle, localPlayerP+offsets.M_iTeamNum, &localTeam) != nil {
		return entities
	}

	needAllBones := SkeletonRendering || BodyHighlightRendering

	for i := 0; i < maxPlayers; i++ {
		entityController, err := readController(procHandle, entityList, i)
		if err != nil || entityController == 0 {
			continue
		}

		pawnHandle, err := readPawnHandle(procHandle, entityController, offsets)
		if err != nil {
			continue
		}

		entityPawn, err := readEntityFromList(procHandle, entityList, pawnHandle)
		if err != nil || entityPawn == 0 || entityPawn == localPlayerP {
			continue
		}

		var lifeState uint8
		if readSafe(procHandle, entityPawn+offsets.M_lifeState, &lifeState) != nil || lifeState != lifeStateAlive {
			continue
		}

		var teamNum int32
		if readSafe(procHandle, entityPawn+offsets.M_iTeamNum, &teamNum) != nil || teamNum < 2 || teamNum > 3 {
			continue
		}
		if TeamCheck && teamNum == localTeam {
			continue
		}

		var health int32
		if readSafe(procHandle, entityPawn+offsets.M_iHealth, &health) != nil || health < 1 || health > 100 {
			continue
		}

		gameScene, err := readPtr(procHandle, entityPawn+offsets.M_pGameSceneNode)
		if err != nil || gameScene == 0 {
			continue
		}

		var dormant bool
		if readSafe(procHandle, gameScene+offsets.M_bDormant, &dormant) == nil && dormant {
			continue
		}

		var entityOrigin Vector3
		originOK := offsets.M_vecAbsOrigin != 0 &&
			readSafe(procHandle, gameScene+offsets.M_vecAbsOrigin, &entityOrigin) == nil
		if !originOK {
			if readSafe(procHandle, entityPawn+offsets.M_vOldOrigin, &entityOrigin) != nil {
				continue
			}
		}

		name := readPlayerName(procHandle, entityController, offsets)

		var entityBones map[string]Vector2
		var entityHead Vector3
		hasHead := false
		boneArrayAddr := gameScene + offsets.M_modelState + boneArrayDelta
		if boneArray, err := readPtr(procHandle, boneArrayAddr); err == nil && boneArray != 0 {
			entityBones, entityHead, hasHead = readBones(procHandle, boneArray, sp, needAllBones)
		}
		if entityBones == nil {
			entityBones = make(map[string]Vector2)
		}

		screenPosFeetX, screenPosFeetY, feetOK := sp.worldToScreen(entityOrigin)
		if !feetOK && hasHead {
			screenPosFeetX, screenPosFeetY, feetOK = sp.worldToScreen(entityHead)
		}
		if !feetOK {
			continue
		}

		entityBoxTopVec := Vector3{entityOrigin.X, entityOrigin.Y, entityOrigin.Z + 72}
		_, screenPosBoxTop, topOK := sp.worldToScreen(entityBoxTopVec)
		if !topOK {
			screenPosBoxTop = screenPosFeetY - 80
		}

		var screenPosHeadX, screenPosHeadTopY, screenPosHeadBottomY float32
		if hasHead {
			entityHeadTop := Vector3{entityHead.X, entityHead.Y, entityHead.Z + 7}
			entityHeadBottom := Vector3{entityHead.X, entityHead.Y, entityHead.Z - 5}
			screenPosHeadX, screenPosHeadTopY, _ = sp.worldToScreen(entityHeadTop)
			_, screenPosHeadBottomY, _ = sp.worldToScreen(entityHeadBottom)
		} else {
			screenPosHeadX = screenPosFeetX
			screenPosHeadTopY = screenPosBoxTop
			screenPosHeadBottomY = screenPosBoxTop + 12
		}

		boxHeight := screenPosFeetY - screenPosBoxTop
		if boxHeight < 8 {
			boxHeight = 40
		}
		boxHalfW := boxHeight / 4
		if boxHalfW < 12 {
			boxHalfW = 12
		}

		entities = append(entities, Entity{
			PawnAddr: entityPawn,
			Health:   health,
			Team:     teamNum,
			Name:     name,
			Distance: entityOrigin.Dist(localPlayerSceneOrigin),
			Position: Vector2{screenPosFeetX, screenPosFeetY},
			Bones:    entityBones,
			HeadPos:  Vector3{screenPosHeadX, screenPosHeadTopY, screenPosHeadBottomY},
			Rect:     Rectangle{screenPosBoxTop, screenPosFeetX - boxHalfW, screenPosFeetX + boxHalfW, screenPosFeetY},
		})
	}

	return entities
}

func readViewProjection(procHandle windows.Handle, clientDll uintptr, offsets Offset, width, height float32) screenProjector {
	var view [16]float32
	if readSafe(procHandle, clientDll+offsets.DwViewMatrix, &view) != nil {
		return screenProjector{}
	}
	return newScreenProjector(width, height, view)
}
