package main

import (
	"fmt"
	"math"
	"unsafe"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

func teamColorFill(team int32) uintptr {
	if team == 2 {
		return uintptr(win.RGB(210, 95, 75))
	}
	return uintptr(win.RGB(95, 90, 225))
}

func teamColorRef(team int32) uintptr {
	if team == 2 {
		return uintptr(win.RGB(255, 142, 120))
	}
	return uintptr(win.RGB(122, 120, 255))
}

func teamColorDim(team int32) uintptr {
	if team == 2 {
		return uintptr(win.RGB(72, 32, 24))
	}
	return uintptr(win.RGB(32, 28, 72))
}

func healthColor(hp int32) uintptr {
	if hp > 66 {
		return uintptr(win.RGB(52, 211, 153))
	}
	if hp > 33 {
		return uintptr(win.RGB(250, 204, 21))
	}
	return uintptr(win.RGB(239, 68, 68))
}

func fillRectColor(hdc win.HDC, left, top, right, bottom int32, color uintptr) {
	if right <= left || bottom <= top {
		return
	}
	brush, _, _ := createSolidBrush.Call(color)
	defer win.DeleteObject(win.HGDIOBJ(brush))
	rect := &win.RECT{Left: left, Top: top, Right: right, Bottom: bottom}
	fillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(rect)), brush)
}

func drawLine(hdc win.HDC, pen uintptr, x1, y1, x2, y2 int32) {
	win.SelectObject(hdc, win.HGDIOBJ(pen))
	win.MoveToEx(hdc, int(x1), int(y1), nil)
	win.LineTo(hdc, x2, y2)
}

func drawCornerBox(hdc win.HDC, rect Rectangle, color uintptr, thickness int32) {
	pen, _, _ := createPen.Call(win.PS_SOLID, uintptr(thickness), color)
	defer win.DeleteObject(win.HGDIOBJ(pen))

	w := rect.Right - rect.Left
	cl := float32(math.Min(float64(w*0.22), 14))
	if cl < 6 {
		cl = 6
	}

	l, t, r, b := int32(rect.Left), int32(rect.Top), int32(rect.Right), int32(rect.Bottom)
	c := int32(cl)

	// top-left
	drawLine(hdc, pen, l, t, l+c, t)
	drawLine(hdc, pen, l, t, l, t+c)
	// top-right
	drawLine(hdc, pen, r, t, r-c, t)
	drawLine(hdc, pen, r, t, r, t+c)
	// bottom-left
	drawLine(hdc, pen, l, b, l+c, b)
	drawLine(hdc, pen, l, b, l, b-c)
	// bottom-right
	drawLine(hdc, pen, r, b, r-c, b)
	drawLine(hdc, pen, r, b, r, b-c)
}

type bodySegment struct {
	from, to string
	width    float32
}

var bodyHighlightSegments = []bodySegment{
	{"head", "neck_0", 0.44},
	{"neck_0", "spine_1", 0.40},
	{"spine_1", "spine_2", 0.42},
	{"spine_2", "pelvis", 0.40},
	{"pelvis", "leg_upper_L", 0.22},
	{"leg_upper_L", "leg_lower_L", 0.19},
	{"leg_lower_L", "ankle_L", 0.16},
	{"pelvis", "leg_upper_R", 0.22},
	{"leg_upper_R", "leg_lower_R", 0.19},
	{"leg_lower_R", "ankle_R", 0.16},
	{"spine_1", "arm_upper_L", 0.18},
	{"arm_upper_L", "arm_lower_L", 0.16},
	{"arm_lower_L", "hand_L", 0.14},
	{"spine_1", "arm_upper_R", 0.18},
	{"arm_upper_R", "arm_lower_R", 0.16},
	{"arm_lower_R", "hand_R", 0.14},
}

func segmentHalfWidth(ax, ay, bx, by, ratio float32) float32 {
	dx := bx - ax
	dy := by - ay
	length := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	hw := length * ratio
	if hw < 3 {
		hw = 3
	}
	if hw > 24 {
		hw = 24
	}
	return hw
}

func fillCapsule(hdc win.HDC, ax, ay, bx, by, halfW float32, brush uintptr) {
	dx := bx - ax
	dy := by - ay
	length := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if length < 0.5 {
		return
	}
	nx := -dy / length * halfW
	ny := dx / length * halfW

	pts := []win.POINT{
		{X: int32(ax + nx), Y: int32(ay + ny)},
		{X: int32(ax - nx), Y: int32(ay - ny)},
		{X: int32(bx - nx), Y: int32(by - ny)},
		{X: int32(bx + nx), Y: int32(by + ny)},
	}
	oldBrush := win.SelectObject(hdc, win.HGDIOBJ(brush))
	oldPen := win.SelectObject(hdc, win.GetStockObject(win.NULL_PEN))
	polygon.Call(uintptr(hdc), uintptr(unsafe.Pointer(&pts[0])), 4)
	win.SelectObject(hdc, oldPen)
	win.SelectObject(hdc, oldBrush)
}

func fillJointCircle(hdc win.HDC, cx, cy, radius float32, brush uintptr) {
	ir := int32(radius)
	if ir < 2 {
		ir = 2
	}
	icx, icy := int32(cx), int32(cy)
	oldBrush := win.SelectObject(hdc, win.HGDIOBJ(brush))
	oldPen := win.SelectObject(hdc, win.GetStockObject(win.NULL_PEN))
	win.Ellipse(hdc, icx-ir, icy-ir, icx+ir, icy+ir)
	win.SelectObject(hdc, oldPen)
	win.SelectObject(hdc, oldBrush)
}

func drawBodyHighlight(hdc win.HDC, bones map[string]Vector2, team int32) {
	if len(bones) < 10 {
		return
	}

	fillBrush, _, _ := createSolidBrush.Call(teamColorFill(team))
	defer win.DeleteObject(win.HGDIOBJ(fillBrush))

	for _, seg := range bodyHighlightSegments {
		a, okA := bones[seg.from]
		b, okB := bones[seg.to]
		if !okA || !okB {
			continue
		}
		hw := segmentHalfWidth(a.X, a.Y, b.X, b.Y, seg.width)
		fillCapsule(hdc, a.X, a.Y, b.X, b.Y, hw, fillBrush)
		fillJointCircle(hdc, a.X, a.Y, hw*0.85, fillBrush)
		fillJointCircle(hdc, b.X, b.Y, hw*0.85, fillBrush)
	}

	if head, ok := bones["head"]; ok {
		r := float32(9)
		if neck, ok2 := bones["neck_0"]; ok2 {
			r = segmentHalfWidth(head.X, head.Y, neck.X, neck.Y, 0.48) * 1.2
		}
		fillJointCircle(hdc, head.X, head.Y, r, fillBrush)
	}
}

func drawModernHealthBar(hdc win.HDC, rect Rectangle, hp int32) {
	barX := int32(rect.Left) - 6
	top := int32(rect.Top)
	bottom := int32(rect.Bottom) + 1
	height := bottom - top
	if height < 4 {
		return
	}

	bgBrush, _, _ := createSolidBrush.Call(0x00202028)
	defer win.DeleteObject(win.HGDIOBJ(bgBrush))
	bgRect := &win.RECT{Left: barX - 1, Top: top - 1, Right: barX + 4, Bottom: bottom + 1}
	fillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(bgRect)), bgBrush)

	fillH := int32(float64(height) * float64(hp) / 100.0)
	if fillH < 1 && hp > 0 {
		fillH = 1
	}
	fillTop := bottom - fillH
	fillBrush, _, _ := createSolidBrush.Call(healthColor(hp))
	defer win.DeleteObject(win.HGDIOBJ(fillBrush))
	hpRect := &win.RECT{Left: barX, Top: fillTop, Right: barX + 3, Bottom: bottom}
	fillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(hpRect)), fillBrush)
}

func drawModernHeadMarker(hdc win.HDC, headPos Vector3, accent uintptr) {
	cx := int32(headPos.X)
	cy := int32((headPos.Y + headPos.Z) / 2)
	r := int32((headPos.Z - headPos.Y) / 2)
	if r < 4 {
		r = 6
	}
	if r > 18 {
		r = 18
	}

	outerPen, _, _ := createPen.Call(win.PS_SOLID, 1, accent)
	defer win.DeleteObject(win.HGDIOBJ(outerPen))
	oldBrush := win.SelectObject(hdc, win.GetStockObject(win.NULL_BRUSH))
	win.SelectObject(hdc, win.HGDIOBJ(outerPen))
	win.Ellipse(hdc, cx-r-1, cy-r-1, cx+r+1, cy+r+1)

	innerPen, _, _ := createPen.Call(win.PS_SOLID, 2, accent)
	defer win.DeleteObject(win.HGDIOBJ(innerPen))
	win.SelectObject(hdc, win.HGDIOBJ(innerPen))
	win.Ellipse(hdc, cx-r, cy-r, cx+r, cy+r)
	win.SelectObject(hdc, oldBrush)
}

func drawModernSkeleton(hdc win.HDC, bones map[string]Vector2, color uintptr) {
	if len(bones) < 10 {
		return
	}
	pen, _, _ := createPen.Call(win.PS_SOLID, 2, color)
	defer win.DeleteObject(win.HGDIOBJ(pen))
	win.SelectObject(hdc, win.HGDIOBJ(pen))

	win.MoveToEx(hdc, int(bones["head"].X), int(bones["head"].Y), nil)
	win.LineTo(hdc, int32(bones["neck_0"].X), int32(bones["neck_0"].Y))
	win.LineTo(hdc, int32(bones["spine_1"].X), int32(bones["spine_1"].Y))
	win.LineTo(hdc, int32(bones["spine_2"].X), int32(bones["spine_2"].Y))
	win.LineTo(hdc, int32(bones["pelvis"].X), int32(bones["pelvis"].Y))
	win.LineTo(hdc, int32(bones["leg_upper_L"].X), int32(bones["leg_upper_L"].Y))
	win.LineTo(hdc, int32(bones["leg_lower_L"].X), int32(bones["leg_lower_L"].Y))
	win.LineTo(hdc, int32(bones["ankle_L"].X), int32(bones["ankle_L"].Y))
	win.MoveToEx(hdc, int(bones["pelvis"].X), int(bones["pelvis"].Y), nil)
	win.LineTo(hdc, int32(bones["leg_upper_R"].X), int32(bones["leg_upper_R"].Y))
	win.LineTo(hdc, int32(bones["leg_lower_R"].X), int32(bones["leg_lower_R"].Y))
	win.LineTo(hdc, int32(bones["ankle_R"].X), int32(bones["ankle_R"].Y))
	win.MoveToEx(hdc, int(bones["spine_1"].X), int(bones["spine_1"].Y), nil)
	win.LineTo(hdc, int32(bones["arm_upper_L"].X), int32(bones["arm_upper_L"].Y))
	win.LineTo(hdc, int32(bones["arm_lower_L"].X), int32(bones["arm_lower_L"].Y))
	win.LineTo(hdc, int32(bones["hand_L"].X), int32(bones["hand_L"].Y))
	win.MoveToEx(hdc, int(bones["spine_1"].X), int(bones["spine_1"].Y), nil)
	win.LineTo(hdc, int32(bones["arm_upper_R"].X), int32(bones["arm_upper_R"].Y))
	win.LineTo(hdc, int32(bones["arm_lower_R"].X), int32(bones["arm_lower_R"].Y))
	win.LineTo(hdc, int32(bones["hand_R"].X), int32(bones["hand_R"].Y))
}

func drawNameTag(hdc win.HDC, rect Rectangle, name string) {
	textW := int32(len(name) * 7)
	if textW < 40 {
		textW = 40
	}
	cx := int32(rect.Left+rect.Right) / 2
	tagLeft := cx - textW/2 - 4
	tagRight := cx + textW/2 + 4
	tagTop := int32(rect.Top) - 20
	tagBottom := int32(rect.Top) - 4

	bgBrush, _, _ := createSolidBrush.Call(0x00181822)
	defer win.DeleteObject(win.HGDIOBJ(bgBrush))
	tagRect := &win.RECT{Left: tagLeft, Top: tagTop, Right: tagRight, Bottom: tagBottom}
	fillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(tagRect)), bgBrush)

	text, _ := windows.UTF16PtrFromString(name)
	win.SetBkMode(hdc, win.TRANSPARENT)
	win.SetTextColor(hdc, win.RGB(240, 240, 245))
	setTextAlign.Call(uintptr(hdc), 0x00000002|0x00000001)
	win.TextOut(hdc, cx, tagTop+2, text, int32(len(name)))
}

func drawHealthText(hdc win.HDC, rect Rectangle, hp int32) {
	hpText := fmt.Sprintf("%d", hp)
	text, _ := windows.UTF16PtrFromString(hpText)
	win.SetBkMode(hdc, win.TRANSPARENT)
	win.SetTextColor(hdc, win.COLORREF(healthColor(hp)))
	setTextAlign.Call(uintptr(hdc), 0x00000002)
	y := int32(rect.Top)
	if HealthBarRendering {
		height := int32(rect.Bottom) - int32(rect.Top)
		fillH := int32(float64(height) * float64(hp) / 100.0)
		y = int32(rect.Bottom) - fillH
	}
	win.TextOut(hdc, int32(rect.Left)-10, y, text, int32(len(hpText)))
}

func renderEntity(hdc win.HDC, entity Entity) {
	accent := teamColorRef(entity.Team)

	if BodyHighlightRendering {
		drawBodyHighlight(hdc, entity.Bones, entity.Team)
	}
	if BoxRendering {
		drawCornerBox(hdc, entity.Rect, accent, 2)
	}
	if SkeletonRendering {
		drawModernSkeleton(hdc, entity.Bones, accent)
	}
	if HeadCircle {
		drawModernHeadMarker(hdc, entity.HeadPos, accent)
	}
	if HealthBarRendering {
		drawModernHealthBar(hdc, entity.Rect, entity.Health)
	}
	if HealthTextRendering {
		drawHealthText(hdc, entity.Rect, entity.Health)
	}
	if NameRendering {
		drawNameTag(hdc, entity.Rect, entity.Name)
	}
}
