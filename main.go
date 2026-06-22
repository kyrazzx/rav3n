package main

import (
	"fmt"
	"log"
	"math"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

const (
	stillActive = 259
	targetFPS   = 120
)

var (
	TeamCheck           = true
	HeadCircle          = true
	SkeletonRendering   = true
	BoxRendering        = true
	NameRendering       = true
	HealthBarRendering  = true
	HealthTextRendering = true
	FrameDelay          int32   = 1
	AimbotEnabled       = true
	AimbotFOV           float32 = 100.0
	AimbotKey           int32   = 0x06
	AimbotSmoothing     float32 = 15.0
	AimbotTarget        string  = "head"
	RecoilEnabled       = false
	RecoilStartBullet   int32   = 1
	RecoilXAxis         float32 = 0.00
	RecoilYAxis         float32 = 2.00
	RecoilSmooth        float32 = 1.00

	shotsFired int32 = 0
)

var (
	user32                     = windows.NewLazySystemDLL("user32.dll")
	gdi32                      = windows.NewLazySystemDLL("gdi32.dll")
	getSystemMetrics           = user32.NewProc("GetSystemMetrics")
	setLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")
	showCursor                 = user32.NewProc("ShowCursor")
	fillRect                   = user32.NewProc("FillRect")
	setTextAlign               = gdi32.NewProc("SetTextAlign")
	createFont                 = gdi32.NewProc("CreateFontW")
	createSolidBrush           = gdi32.NewProc("CreateSolidBrush")
	createPen                  = gdi32.NewProc("CreatePen")
	getAsyncKeyState           = user32.NewProc("GetAsyncKeyState")
	mouseEvent                 = user32.NewProc("mouse_event")
)

func init() { runtime.LockOSThread() }

func logAndSleep(message string, err error) {
	log.Printf("%s: %v\n", message, err)
	time.Sleep(5 * time.Second)
}

func drawSkeleton(hdc win.HDC, pen uintptr, bones map[string]Vector2) {
	if len(bones) < 10 {
		return
	}
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

func renderEntityInfo(hdc win.HDC, tPen, gPen, oPen, hPen uintptr, rect Rectangle, hp int32, name string, headPos Vector3) {
	if BoxRendering {
		win.SelectObject(hdc, win.HGDIOBJ(tPen))
		win.MoveToEx(hdc, int(rect.Left), int(rect.Top), nil)
		win.LineTo(hdc, int32(rect.Right), int32(rect.Top))
		win.LineTo(hdc, int32(rect.Right), int32(rect.Bottom))
		win.LineTo(hdc, int32(rect.Left), int32(rect.Bottom))
		win.LineTo(hdc, int32(rect.Left), int32(rect.Top))
		win.SelectObject(hdc, win.HGDIOBJ(oPen))
		win.MoveToEx(hdc, int(rect.Left)-1, int(rect.Top)-1, nil)
		win.LineTo(hdc, int32(rect.Right)-1, int32(rect.Top)+1)
		win.LineTo(hdc, int32(rect.Right)+1, int32(rect.Bottom)+1)
		win.LineTo(hdc, int32(rect.Left)+1, int32(rect.Bottom)-1)
		win.LineTo(hdc, int32(rect.Left)-1, int32(rect.Top)-1)
		win.MoveToEx(hdc, int(rect.Left)+1, int(rect.Top)+1, nil)
		win.LineTo(hdc, int32(rect.Right)+1, int32(rect.Top)-1)
		win.LineTo(hdc, int32(rect.Right)-1, int32(rect.Bottom)-1)
		win.LineTo(hdc, int32(rect.Left)-1, int32(rect.Bottom)+1)
		win.LineTo(hdc, int32(rect.Left)+1, int32(rect.Top)+1)
	}
	if HeadCircle {
		radius := int32((headPos.Z - headPos.Y) / 2)
		oldBrush := win.SelectObject(hdc, win.GetStockObject(win.NULL_BRUSH))
		win.SelectObject(hdc, win.HGDIOBJ(hPen))
		win.Ellipse(hdc, int32(headPos.X)-radius, int32(headPos.Y), int32(headPos.X)+radius, int32(headPos.Z))
		win.SelectObject(hdc, oldBrush)
	}
	if HealthBarRendering {
		win.SelectObject(hdc, win.HGDIOBJ(gPen))
		win.MoveToEx(hdc, int(rect.Left)-4, int(rect.Bottom)+1-int(float64(int(rect.Bottom)+1-int(rect.Top))*float64(hp)/100.0), nil)
		win.LineTo(hdc, int32(rect.Left)-4, int32(rect.Bottom)+1)
		win.SelectObject(hdc, win.HGDIOBJ(oPen))
		win.MoveToEx(hdc, int(rect.Left)-5, int(rect.Top)-1, nil)
		win.LineTo(hdc, int32(rect.Left)-3, int32(rect.Top)-1)
		win.LineTo(hdc, int32(rect.Left)-3, int32(rect.Bottom)+1)
		win.LineTo(hdc, int32(rect.Left)-5, int32(rect.Bottom)+1)
		win.LineTo(hdc, int32(rect.Left)-5, int32(rect.Top)-1)
	}
	if HealthTextRendering {
		hpText := fmt.Sprintf("%d", hp)
		text, _ := windows.UTF16PtrFromString(hpText)
		win.SetTextColor(hdc, win.RGB(0, 255, 50))
		setTextAlign.Call(uintptr(hdc), 0x00000002)
		if HealthBarRendering {
			win.TextOut(hdc, int32(rect.Left)-8, int32(int(rect.Bottom)+1-int(float64(int(rect.Bottom)+1-int(rect.Top))*float64(hp)/100.0)), text, int32(len(hpText)))
		} else {
			win.TextOut(hdc, int32(rect.Left)-4, int32(rect.Top), text, int32(len(hpText)))
		}
	}
	if NameRendering {
		text, _ := windows.UTF16PtrFromString(name)
		win.SetTextColor(hdc, win.RGB(255, 255, 255))
		setTextAlign.Call(uintptr(hdc), 0x00000006)
		win.TextOut(hdc, int32(rect.Left)+int32((rect.Right-rect.Left)/2), int32(rect.Top)-14, text, int32(len(name)))
	}
}

func windowProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if msg == win.WM_DESTROY {
		win.PostQuitMessage(0)
		return 0
	}
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func initWindow(screenWidth, screenHeight uintptr) win.HWND {
	className, _ := windows.UTF16PtrFromString("OverlayWindow")
	windowTitle, _ := windows.UTF16PtrFromString("Overlay")
	wc := win.WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(win.WNDCLASSEX{})),
		Style:         win.CS_HREDRAW | win.CS_VREDRAW,
		LpfnWndProc:   syscall.NewCallback(windowProc),
		HInstance:     win.GetModuleHandle(nil),
		HbrBackground: win.COLOR_WINDOW,
		LpszClassName: className,
	}
	if win.RegisterClassEx(&wc) == 0 {
		logAndSleep("Error registering window class", fmt.Errorf("%v", win.GetLastError()))
		return 0
	}
	hwnd := win.CreateWindowEx(
		win.WS_EX_TOPMOST|win.WS_EX_NOACTIVATE|win.WS_EX_LAYERED|win.WS_EX_TRANSPARENT,
		className, windowTitle, win.WS_POPUP, 0, 0, int32(screenWidth), int32(screenHeight),
		0, 0, win.GetModuleHandle(nil), nil)
	if hwnd == 0 {
		logAndSleep("Error creating window", fmt.Errorf("%v", win.GetLastError()))
		return 0
	}
	setLayeredWindowAttributes.Call(uintptr(hwnd), 0x000000, 0, 0x00000001)
	showCursor.Call(0)
	win.ShowWindow(hwnd, win.SW_SHOWDEFAULT)
	return hwnd
}

func aimbot(entities []Entity, screenWidth, screenHeight uintptr) {
	state, _, _ := getAsyncKeyState.Call(uintptr(AimbotKey))
	if state&0x8000 == 0 {
		return
	}

	var closestDistance float32 = 10000.0
	var targetEntity *Entity
	crosshairX, crosshairY := float32(screenWidth/2), float32(screenHeight/2)

	for i := range entities {
		entity := &entities[i]
		targetBonePos, ok := entity.Bones[AimbotTarget]
		if !ok || targetBonePos.X < 0 || targetBonePos.Y < 0 {
			continue
		}
		dx := targetBonePos.X - crosshairX
		dy := targetBonePos.Y - crosshairY
		distance := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		if distance < closestDistance {
			closestDistance = distance
			targetEntity = entity
		}
	}

	if targetEntity != nil && closestDistance < AimbotFOV {
		targetPos := targetEntity.Bones[AimbotTarget]
		moveX := (targetPos.X - crosshairX) / AimbotSmoothing
		moveY := (targetPos.Y - crosshairY) / AimbotSmoothing
		if moveX != 0 || moveY != 0 {
			mouseEvent.Call(0x0001, uintptr(int32(math.Round(float64(moveX)))), uintptr(int32(math.Round(float64(moveY)))), 0, 0)
		}
	}
}

func recoilControl() {
	if !RecoilEnabled {
		return
	}
	state, _, _ := getAsyncKeyState.Call(0x01)
	if state&0x8000 != 0 {
		shotsFired++
		if shotsFired > RecoilStartBullet {
			moveX := RecoilXAxis / RecoilSmooth
			moveY := RecoilYAxis / RecoilSmooth
			mouseEvent.Call(0x0001, uintptr(int32(math.Round(float64(moveX)))), uintptr(int32(math.Round(float64(moveY)))), 0, 0)
		}
	} else {
		shotsFired = 0
	}
}

func main() {
	offsets := loadOffsets()
	go RunGui()

	screenWidth, _, _ := getSystemMetrics.Call(0)
	screenHeight, _, _ := getSystemMetrics.Call(1)
	projector := newScreenProjector(screenWidth, screenHeight)

	hwnd := initWindow(screenWidth, screenHeight)
	if hwnd == 0 {
		return
	}
	defer win.DestroyWindow(hwnd)

	pid, err := findProcessId("cs2.exe")
	if err != nil {
		logAndSleep("Error finding process ID, is cs2.exe running?", err)
		return
	}
	clientDll, err := getModuleBaseAddress(pid, "client.dll")
	if err != nil {
		logAndSleep("Error getting client.dll base address", err)
		return
	}
	procHandle, err := getProcessHandle(pid)
	if err != nil {
		logAndSleep("Error getting process handle", err)
		return
	}
	defer windows.CloseHandle(procHandle)

	hdc := win.GetDC(hwnd)
	if hdc == 0 {
		logAndSleep("Error getting device context", fmt.Errorf("%v", win.GetLastError()))
		return
	}
	defer win.ReleaseDC(hwnd, hdc)

	memhdc := win.CreateCompatibleDC(hdc)
	memBitmap := win.CreateCompatibleBitmap(hdc, int32(screenWidth), int32(screenHeight))
	win.SelectObject(memhdc, win.HGDIOBJ(memBitmap))
	defer func() {
		win.DeleteObject(win.HGDIOBJ(memBitmap))
		win.DeleteDC(memhdc)
	}()

	bgBrush, _, _ := createSolidBrush.Call(0x000000)
	defer win.DeleteObject(win.HGDIOBJ(bgBrush))
	redPen, _, _ := createPen.Call(win.PS_SOLID, 1, 0x7a78ff)
	defer win.DeleteObject(win.HGDIOBJ(redPen))
	greenPen, _, _ := createPen.Call(win.PS_SOLID, 1, 0x7dff78)
	defer win.DeleteObject(win.HGDIOBJ(greenPen))
	bluePen, _, _ := createPen.Call(win.PS_SOLID, 1, 0xff8e78)
	defer win.DeleteObject(win.HGDIOBJ(bluePen))
	bonePen, _, _ := createPen.Call(win.PS_SOLID, 1, 0xffffff)
	defer win.DeleteObject(win.HGDIOBJ(bonePen))
	outlinePen, _, _ := createPen.Call(win.PS_SOLID, 1, 0x000001)
	defer win.DeleteObject(win.HGDIOBJ(outlinePen))
	fovPen, _, _ := createPen.Call(win.PS_SOLID, 1, 0xFFFFFF)
	defer win.DeleteObject(win.HGDIOBJ(fovPen))
	font, _, _ := createFont.Call(12, 0, 0, 0, win.FW_HEAVY, 0, 0, 0, win.DEFAULT_CHARSET, win.OUT_DEFAULT_PRECIS, win.CLIP_DEFAULT_PRECIS, win.DEFAULT_QUALITY, win.DEFAULT_PITCH|win.FF_DONTCARE, 0)
	defer win.DeleteObject(win.HGDIOBJ(font))

	rect := &win.RECT{Left: 0, Top: 0, Right: int32(screenWidth), Bottom: int32(screenHeight)}
	frameDuration := time.Second / targetFPS

	for {
		frameStart := time.Now()

		var exitCode uint32
		if err := windows.GetExitCodeProcess(procHandle, &exitCode); err != nil || exitCode != stillActive {
			fmt.Println("Process cs2.exe not found or has exited. Exiting program.")
			break
		}

		fillRect.Call(uintptr(memhdc), uintptr(unsafe.Pointer(rect)), bgBrush)
		win.SetBkMode(memhdc, win.TRANSPARENT)
		win.SelectObject(memhdc, win.HGDIOBJ(font))

		entities := getEntitiesInfo(procHandle, clientDll, projector, offsets)
		recoilControl()

		if AimbotEnabled {
			oldBrush := win.SelectObject(memhdc, win.GetStockObject(win.NULL_BRUSH))
			win.SelectObject(memhdc, win.HGDIOBJ(fovPen))
			win.Ellipse(memhdc, int32(screenWidth/2)-int32(AimbotFOV), int32(screenHeight/2)-int32(AimbotFOV), int32(screenWidth/2)+int32(AimbotFOV), int32(screenHeight/2)+int32(AimbotFOV))
			win.SelectObject(memhdc, oldBrush)
			aimbot(entities, screenWidth, screenHeight)
		}

		for _, entity := range entities {
			if entity.Distance < 35 {
				continue
			}
			if SkeletonRendering {
				drawSkeleton(memhdc, bonePen, entity.Bones)
			}
			if entity.Team == 2 {
				renderEntityInfo(memhdc, redPen, greenPen, outlinePen, bonePen, entity.Rect, entity.Health, entity.Name, entity.HeadPos)
			} else {
				renderEntityInfo(memhdc, bluePen, greenPen, outlinePen, bonePen, entity.Rect, entity.Health, entity.Name, entity.HeadPos)
			}
		}

		win.BitBlt(hdc, 0, 0, int32(screenWidth), int32(screenHeight), memhdc, 0, 0, win.SRCCOPY)

		elapsed := time.Since(frameStart)
		recordOverlayFrame(float32(elapsed.Milliseconds()))
		minFrame := frameDuration
		if FrameDelay > 0 {
			delay := time.Duration(FrameDelay) * time.Millisecond
			if delay > minFrame {
				minFrame = delay
			}
		}
		if elapsed < minFrame {
			time.Sleep(minFrame - elapsed)
		}
	}
}
