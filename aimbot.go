package main
import (
	"math"
	"time"
)
const (
	aimBoneSamples = 4
	aimTickMs      = 16
)
var (
	aimLockedPawn  uintptr
	aimBoneBuf     [aimBoneSamples]Vector2
	aimBoneCount   int
	aimBoneIdx     int
	lastAimTick    time.Time
	aimOnTarget    bool
	aimbotFOVSq    float32
	aimbotFOVLockSq float32
)
func resetAimState() {
	aimLockedPawn = 0
	aimBoneCount = 0
	aimBoneIdx = 0
	aimOnTarget = false
}
func syncAimbotFOV() {
	aimbotFOVSq = AimbotFOV * AimbotFOV
	aimbotFOVLockSq = aimbotFOVSq * 1.8225
}
func avgBonePos(raw Vector2) Vector2 {
	aimBoneBuf[aimBoneIdx] = raw
	aimBoneIdx = (aimBoneIdx + 1) % aimBoneSamples
	if aimBoneCount < aimBoneSamples {
		aimBoneCount++
	}
	var sx, sy float32
	for i := 0; i < aimBoneCount; i++ {
		sx += aimBoneBuf[i].X
		sy += aimBoneBuf[i].Y
	}
	n := float32(aimBoneCount)
	return Vector2{X: sx / n, Y: sy / n}
}
func resetBoneBuffer(raw Vector2) {
	for i := range aimBoneBuf {
		aimBoneBuf[i] = raw
	}
	aimBoneCount = aimBoneSamples
	aimBoneIdx = 0
}
func isShooting() bool {
	state, _, _ := getAsyncKeyState.Call(0x01)
	return state&0x8000 != 0
}
func findEntityByPawn(entities []Entity, pawn uintptr) *Entity {
	if pawn == 0 {
		return nil
	}
	for i := range entities {
		if entities[i].PawnAddr == pawn {
			return &entities[i]
		}
	}
	return nil
}
func closestEntityInFOV(entities []Entity, crosshairX, crosshairY float32) *Entity {
	var best *Entity
	bestDistSq := aimbotFOVSq
	for i := range entities {
		e := &entities[i]
		pos, ok := e.Bones[AimbotTarget]
		if !ok {
			continue
		}
		dx := pos.X - crosshairX
		dy := pos.Y - crosshairY
		distSq := dx*dx + dy*dy
		if distSq < bestDistSq {
			bestDistSq = distSq
			best = e
		}
	}
	return best
}
func dist2DSq(a, b Vector2) float32 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return dx*dx + dy*dy
}
func resolveAimTarget(entities []Entity, crosshairX, crosshairY float32) (*Entity, Vector2, bool) {
	ch := Vector2{crosshairX, crosshairY}
	if aimLockedPawn != 0 {
		if locked := findEntityByPawn(entities, aimLockedPawn); locked != nil {
			if pos, ok := locked.Bones[AimbotTarget]; ok {
				if dist2DSq(pos, ch) <= aimbotFOVLockSq {
					return locked, pos, true
				}
			}
		}
		aimLockedPawn = 0
		aimBoneCount = 0
	}
	candidate := closestEntityInFOV(entities, crosshairX, crosshairY)
	if candidate == nil {
		return nil, Vector2{}, false
	}
	pos, ok := candidate.Bones[AimbotTarget]
	if !ok {
		return nil, Vector2{}, false
	}
	return candidate, pos, true
}
func moveRatio(smooth float32) float32 {
	switch {
	case smooth <= 1.5:
		return 0.50
	case smooth <= 3:
		return 0.35
	case smooth <= 6:
		return 0.22
	case smooth <= 12:
		return 0.14
	default:
		return 0.08
	}
}
func aimbot(entities []Entity, crosshairX, crosshairY float32) {
	syncAimbotFOV()
	state, _, _ := getAsyncKeyState.Call(uintptr(AimbotKey))
	if state&0x8000 == 0 {
		resetAimState()
		return
	}
	now := time.Now()
	if now.Sub(lastAimTick) < aimTickMs*time.Millisecond {
		return
	}
	lastAimTick = now
	target, rawPos, ok := resolveAimTarget(entities, crosshairX, crosshairY)
	if !ok || target == nil {
		resetAimState()
		return
	}
	if aimLockedPawn != target.PawnAddr {
		aimLockedPawn = target.PawnAddr
		resetBoneBuffer(rawPos)
		aimOnTarget = false
	}
	aimPos := avgBonePos(rawPos)
	dx := aimPos.X - crosshairX
	dy := aimPos.Y - crosshairY
	distSq := dx*dx + dy*dy
	dist := float32(math.Sqrt(float64(distSq)))
	smooth := AimbotSmoothing
	if smooth < 1 {
		smooth = 1
	}
	enterZone := 3.5 + smooth*0.15
	exitZone := enterZone + 2.5
	if aimOnTarget {
		if dist < exitZone {
			return
		}
		aimOnTarget = false
	} else if dist < enterZone {
		aimOnTarget = true
		return
	}
	shooting := isShooting()
	if shooting && dist < enterZone+4 {
		return
	}
	moveX := dx / smooth
	moveY := dy / smooth
	maxMove := dist * moveRatio(smooth)
	if shooting {
		maxMove *= 0.5
	}
	moveMagSq := moveX*moveX + moveY*moveY
	maxMoveSq := maxMove * maxMove
	if moveMagSq > maxMoveSq && moveMagSq > 0 {
		scale := maxMove / float32(math.Sqrt(float64(moveMagSq)))
		moveX *= scale
		moveY *= scale
	}
	ix := int(math.Trunc(float64(moveX)))
	iy := int(math.Trunc(float64(moveY)))
	if ix == 0 && iy == 0 {
		return
	}
	mouseEvent.Call(0x0001, uintptr(int32(ix)), uintptr(int32(iy)), 0, 0)
}
