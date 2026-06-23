package main
import (
	"sync"
	"syscall"
	"time"
	"github.com/lxn/win"
)
var (
	procEnumWindows     = user32.NewProc("EnumWindows")
	procIsWindow        = user32.NewProc("IsWindow")
	enumWindowsCallback  uintptr
	enumWindowsOnce      sync.Once
	enumTargetPID        int
	enumBestHWND         win.HWND
	enumBestArea         int32
	cachedGameHWND       win.HWND
	cachedGamePID        int
	cachedGameCheckedAt  time.Time
	gameWindowCacheTTL   = 750 * time.Millisecond
	lastSyncedArea       gameClientArea
)
func enumWindowsProc(hwnd win.HWND, _ uintptr) uintptr {
	var windowPID uint32
	win.GetWindowThreadProcessId(hwnd, &windowPID)
	if int(windowPID) != enumTargetPID || !win.IsWindowVisible(hwnd) {
		return 1
	}
	area := getGameClientArea(hwnd)
	size := area.width * area.height
	if size > enumBestArea {
		enumBestArea = size
		enumBestHWND = hwnd
	}
	return 1
}

func initEnumWindowsCallback() {
	enumWindowsOnce.Do(func() {
		enumWindowsCallback = syscall.NewCallback(enumWindowsProc)
	})
}

func isWindow(hwnd win.HWND) bool {
	r, _, _ := procIsWindow.Call(uintptr(hwnd))
	return r != 0
}

func findGameWindow(pid int) win.HWND {
	now := time.Now()
	if cachedGamePID == pid && cachedGameHWND != 0 && now.Sub(cachedGameCheckedAt) < gameWindowCacheTTL {
		if isWindow(cachedGameHWND) && win.IsWindowVisible(cachedGameHWND) {
			return cachedGameHWND
		}
	}
	initEnumWindowsCallback()
	enumTargetPID = pid
	enumBestHWND = 0
	enumBestArea = 0
	procEnumWindows.Call(enumWindowsCallback, 0)
	cachedGamePID = pid
	cachedGameHWND = enumBestHWND
	cachedGameCheckedAt = now
	return enumBestHWND
}
type gameClientArea struct {
	x, y, width, height int32
}

func getGameClientArea(hwnd win.HWND) gameClientArea {
	var client win.RECT
	win.GetClientRect(hwnd, &client)
	pt := win.POINT{X: 0, Y: 0}
	win.ClientToScreen(hwnd, &pt)
	return gameClientArea{
		x:      pt.X,
		y:      pt.Y,
		width:  client.Right - client.Left,
		height: client.Bottom - client.Top,
	}
}

func syncOverlayToGame(overlay, game win.HWND) gameClientArea {
	area := getGameClientArea(game)
	if area.width <= 0 || area.height <= 0 {
		return area
	}
	if area == lastSyncedArea {
		return area
	}
	lastSyncedArea = area
	win.SetWindowPos(
		overlay, win.HWND_TOPMOST,
		area.x, area.y, area.width, area.height,
		win.SWP_NOACTIVATE|win.SWP_SHOWWINDOW,
	)
	return area
}
