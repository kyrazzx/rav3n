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
	stillActive        = 259
	targetFPS          = 120
	exitCheckInterval  = 60
)
var (
	TeamCheck           = true
	HeadCircle          = true
	SkeletonRendering   = true
	BoxRendering         = true
	BodyHighlightRendering = true
	NameRendering        = true
	HealthBarRendering  = true
	HealthTextRendering = true
	FrameDelay          int32   = 1
	AimbotEnabled       = true
	AimbotFOV           float32 = 100.0
	AimbotKey           int32   = 0x06
	AimbotSmoothing     float32 = 8.0
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
	setLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")
	showCursor                 = user32.NewProc("ShowCursor")
	fillRect                   = user32.NewProc("FillRect")
	setTextAlign               = gdi32.NewProc("SetTextAlign")
	createFont                 = gdi32.NewProc("CreateFontW")
	createSolidBrush           = gdi32.NewProc("CreateSolidBrush")
	createPen                  = gdi32.NewProc("CreatePen")
	polygon                    = gdi32.NewProc("Polygon")
	getAsyncKeyState           = user32.NewProc("GetAsyncKeyState")
	mouseEvent                 = user32.NewProc("mouse_event")
)
func logAndSleep(message string, err error) {
	log.Printf("%s: %v\n", message, err)
	time.Sleep(5 * time.Second)
}

func windowProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if msg == win.WM_DESTROY {
		win.PostQuitMessage(0)
		return 0
	}
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func initWindow(width, height int32) win.HWND {
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
		className, windowTitle, win.WS_POPUP, 0, 0, width, height,
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

func runOverlay(offsets Offset) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	pid, err := findProcessId("cs2.exe")
	if err != nil {
		logAndSleep("Error finding process ID, is cs2.exe running?", err)
		return
	}
	gameWnd := findGameWindow(pid)
	if gameWnd == 0 {
		logAndSleep("Could not find CS2 window", fmt.Errorf("hwnd not found"))
		return
	}
	area := getGameClientArea(gameWnd)
	if area.width <= 0 || area.height <= 0 {
		area.width, area.height = 1920, 1080
	}
	hwnd := initWindow(area.width, area.height)
	if hwnd == 0 {
		return
	}
	defer win.DestroyWindow(hwnd)
	syncOverlayToGame(hwnd, gameWnd)
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
	memBitmap := win.CreateCompatibleBitmap(hdc, area.width, area.height)
	win.SelectObject(memhdc, win.HGDIOBJ(memBitmap))
	defer func() {
		win.DeleteObject(win.HGDIOBJ(memBitmap))
		win.DeleteDC(memhdc)
	}()
	var overlayW, overlayH = area.width, area.height
	rect := &win.RECT{Left: 0, Top: 0, Right: overlayW, Bottom: overlayH}
	bgBrush, _, _ := createSolidBrush.Call(0x000000)
	defer win.DeleteObject(win.HGDIOBJ(bgBrush))
	font, _, _ := createFont.Call(13, 0, 0, 0, win.FW_SEMIBOLD, 0, 0, 0, win.DEFAULT_CHARSET, win.OUT_DEFAULT_PRECIS, win.CLIP_DEFAULT_PRECIS, win.CLEARTYPE_QUALITY, win.DEFAULT_PITCH|win.FF_DONTCARE, 0)
	defer win.DeleteObject(win.HGDIOBJ(font))
	initOverlayGDI()
	defer destroyOverlayGDI()
	win.SetBkMode(memhdc, win.TRANSPARENT)
	win.SelectObject(memhdc, win.HGDIOBJ(font))
	frameDuration := time.Second / targetFPS
	crosshairX := float32(overlayW) / 2
	crosshairY := float32(overlayH) / 2
	exitCheckCounter := 0
	for {
		frameStart := time.Now()
		exitCheckCounter++
		if exitCheckCounter >= exitCheckInterval {
			exitCheckCounter = 0
			var exitCode uint32
			if err := windows.GetExitCodeProcess(procHandle, &exitCode); err != nil || exitCode != stillActive {
				fmt.Println("Process cs2.exe not found or has exited. Exiting program.")
				break
			}
		}
		if gameWnd = findGameWindow(pid); gameWnd != 0 {
			area = syncOverlayToGame(hwnd, gameWnd)
			if area.width > 0 && area.height > 0 && (area.width != overlayW || area.height != overlayH) {
				overlayW, overlayH = area.width, area.height
				win.DeleteObject(win.HGDIOBJ(memBitmap))
				memBitmap = win.CreateCompatibleBitmap(hdc, overlayW, overlayH)
				win.SelectObject(memhdc, win.HGDIOBJ(memBitmap))
				rect.Right = overlayW
				rect.Bottom = overlayH
				crosshairX = float32(overlayW) / 2
				crosshairY = float32(overlayH) / 2
			}
		}
		fillRect.Call(uintptr(memhdc), uintptr(unsafe.Pointer(rect)), bgBrush)
		projector := readViewProjection(procHandle, clientDll, offsets, float32(overlayW), float32(overlayH))
		entities := getEntitiesInfo(procHandle, clientDll, projector, offsets)
		recoilControl()
		if AimbotEnabled {
			aimbot(entities, crosshairX, crosshairY)
		}
		for _, entity := range entities {
			renderEntity(memhdc, entity)
		}
		win.BitBlt(hdc, 0, 0, overlayW, overlayH, memhdc, 0, 0, win.SRCCOPY)
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

func main() {
	offsets := loadOffsets()
	go runOverlay(offsets)
	RunGui()
}
