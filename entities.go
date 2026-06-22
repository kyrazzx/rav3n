package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"unicode/utf8"

	"golang.org/x/sys/windows"
)

const (
	maxPlayers      = 64
	boneStride      = 32
	maxBoneIndex    = 28
	boneBufferSize  = (maxBoneIndex + 1) * boneStride
	lifeStateAlive  = 256
)

var boneIndices = map[string]int{
	"head": 6, "neck_0": 5, "spine_1": 4, "spine_2": 2, "pelvis": 0,
	"arm_upper_L": 8, "arm_lower_L": 9, "hand_L": 10,
	"arm_upper_R": 13, "arm_lower_R": 14, "hand_R": 15,
	"leg_upper_L": 22, "leg_lower_L": 23, "ankle_L": 24,
	"leg_upper_R": 25, "leg_lower_R": 26, "ankle_R": 27,
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
	halfW, halfH float32
	fullW, fullH float32
}

func newScreenProjector(width, height uintptr) screenProjector {
	w := float32(width)
	h := float32(height)
	return screenProjector{halfW: w / 2, halfH: h / 2, fullW: w, fullH: h}
}

func (sp screenProjector) worldToScreen(viewMatrix Matrix, position Vector3) (float32, float32) {
	screenX := viewMatrix[0][0]*position.X + viewMatrix[0][1]*position.Y + viewMatrix[0][2]*position.Z + viewMatrix[0][3]
	screenY := viewMatrix[1][0]*position.X + viewMatrix[1][1]*position.Y + viewMatrix[1][2]*position.Z + viewMatrix[1][3]
	w := viewMatrix[3][0]*position.X + viewMatrix[3][1]*position.Y + viewMatrix[3][2]*position.Z + viewMatrix[3][3]
	if w < 0.01 {
		return -1, -1
	}
	invw := 1.0 / w
	screenX *= invw
	screenY *= invw
	x := sp.halfW + 0.5*screenX*sp.fullW + 0.5
	y := sp.halfH - 0.5*screenY*sp.fullH + 0.5
	return x, y
}

func readPlayerName(procHandle windows.Handle, controller uintptr, offsets Offset) (string, error) {
	nameBytes, err := readAt(procHandle, controller+offsets.M_iszPlayerName, 128)
	if err != nil {
		return "", err
	}
	end := 0
	for end < len(nameBytes) && nameBytes[end] != 0 {
		end++
	}
	if end == 0 {
		return "", fmt.Errorf("empty player name")
	}
	if utf8.Valid(nameBytes[:end]) {
		return string(nameBytes[:end]), nil
	}
	return "", fmt.Errorf("invalid player name encoding")
}

func readBones(procHandle windows.Handle, boneArray uintptr, viewMatrix Matrix, sp screenProjector, allBones bool) (map[string]Vector2, Vector3, bool) {
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
		if boneName == "head" {
			headPos = pos
			hasHead = true
		}
		x, y := sp.worldToScreen(viewMatrix, pos)
		bones[boneName] = Vector2{X: x, Y: y}
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

	var viewMatrix Matrix
	if readSafe(procHandle, clientDll+offsets.DwViewMatrix, &viewMatrix) != nil {
		return entities
	}

	needAllBones := SkeletonRendering

	for i := 1; i <= maxPlayers; i++ {
		entityController, err := readController(procHandle, entityList, i)
		if err != nil || entityController == 0 {
			continue
		}

		var pawnHandle uint32
		if readSafe(procHandle, entityController+offsets.M_hPlayerPawn, &pawnHandle) != nil || pawnHandle == 0 {
			continue
		}

		entityPawn, err := readEntityFromList(procHandle, entityList, pawnHandle)
		if err != nil || entityPawn == 0 || entityPawn == localPlayerP {
			continue
		}

		var pawnAlive bool
		if readSafe(procHandle, entityController+offsets.M_bPawnIsAlive, &pawnAlive) == nil && !pawnAlive {
			continue
		}

		var lifeState int32
		if readSafe(procHandle, entityPawn+offsets.M_lifeState, &lifeState) != nil || lifeState != lifeStateAlive {
			continue
		}

		var teamNum int32
		if readSafe(procHandle, entityPawn+offsets.M_iTeamNum, &teamNum) != nil || teamNum == 0 {
			continue
		}
		if TeamCheck && teamNum == localTeam {
			continue
		}

		var health int32
		if readSafe(procHandle, entityPawn+offsets.M_iHealth, &health) != nil || health < 1 || health > 100 {
			continue
		}

		name, err := readPlayerName(procHandle, entityController, offsets)
		if err != nil {
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
		if readSafe(procHandle, entityPawn+offsets.M_vOldOrigin, &entityOrigin) != nil {
			continue
		}

		var entityBones map[string]Vector2
		var entityHead Vector3
		boneArray, err := readPtr(procHandle, gameScene+offsets.M_boneArray)
		if err != nil || boneArray == 0 {
			continue
		}
		var ok bool
		entityBones, entityHead, ok = readBones(procHandle, boneArray, viewMatrix, sp, needAllBones)
		if !ok {
			continue
		}

		entityHeadTop := Vector3{entityHead.X, entityHead.Y, entityHead.Z + 7}
		entityHeadBottom := Vector3{entityHead.X, entityHead.Y, entityHead.Z - 5}
		screenPosHeadX, screenPosHeadTopY := sp.worldToScreen(viewMatrix, entityHeadTop)
		_, screenPosHeadBottomY := sp.worldToScreen(viewMatrix, entityHeadBottom)
		screenPosFeetX, screenPosFeetY := sp.worldToScreen(viewMatrix, entityOrigin)
		entityBoxTopVec := Vector3{entityOrigin.X, entityOrigin.Y, entityOrigin.Z + 70}
		_, screenPosBoxTop := sp.worldToScreen(viewMatrix, entityBoxTopVec)

		if screenPosHeadX <= -1 || screenPosFeetY <= -1 ||
			screenPosHeadX >= sp.fullW || screenPosHeadTopY >= sp.fullH {
			continue
		}

		boxHeight := screenPosFeetY - screenPosBoxTop
		entities = append(entities, Entity{
			Health:   health,
			Team:     teamNum,
			Name:     name,
			Distance: entityOrigin.Dist(localPlayerSceneOrigin),
			Position: Vector2{screenPosFeetX, screenPosFeetY},
			Bones:    entityBones,
			HeadPos:  Vector3{screenPosHeadX, screenPosHeadTopY, screenPosHeadBottomY},
			Rect:     Rectangle{screenPosBoxTop, screenPosFeetX - boxHeight/4, screenPosFeetX + boxHeight/4, screenPosFeetY},
		})
	}

	return entities
}
