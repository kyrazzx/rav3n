package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unicode"
	"unsafe"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

const STILL_ACTIVE = 259
const targetFPS = 120

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
	AimbotKey           int32   = 0x06 // VK_XBUTTON2 (Mouse 5) by default
	AimbotSmoothing     float32 = 15.0
	AimbotTarget        string  = "head" // Default aimbot target
	RecoilEnabled       = false
	RecoilStartBullet   int32   = 1
	RecoilXAxis         float32 = 0.00
	RecoilYAxis         float32 = 2.00
	RecoilSmooth        float32 = 1.00

	shotsFired int32 = 0
)

type CombinedOffsets struct {
	DwEntityList           uintptr `json:"dwEntityList"`
	DwLocalPlayerPawn      uintptr `json:"dwLocalPlayerPawn"`
	DwViewMatrix           uintptr `json:"dwViewMatrix"`
	M_hPlayerPawn          uintptr `json:"m_hPlayerPawn"`
	M_iHealth              uintptr `json:"m_iHealth"`
	M_lifeState            uintptr `json:"m_lifeState"`
	M_iTeamNum             uintptr `json:"m_iTeamNum"`
	M_vOldOrigin           uintptr `json:"m_vOldOrigin"`
	M_pGameSceneNode       uintptr `json:"m_pGameSceneNode"`
	M_modelState           uintptr `json:"m_modelState"`
	M_nodeToWorld          uintptr `json:"m_nodeToWorld"`
	M_sSanitizedPlayerName uintptr `json:"m_sSanitizedPlayerName"`
	M_boneArray            uintptr `json:"m_boneArray"`
}

func getValue(data map[string]interface{}, key string) uintptr {
	if value, exists := data[key]; exists && value != nil {
		if floatValue, ok := value.(float64); ok {
			return uintptr(floatValue)
		}
		log.Fatalf("Key '%s' is not of expected type (float64)", key)
	}
	log.Fatalf("Key '%s' does not exist or is nil", key)
	return 0
}

func getNestedFieldValue(data map[string]interface{}, dllKey, classKey, fieldKey string) uintptr {
	dllData, exists := data[dllKey]
	if !exists {
		log.Fatalf("Main key '%s' does not exist in JSON", dllKey)
	}
	dllMap, ok := dllData.(map[string]interface{})
	if !ok {
		log.Fatalf("Value of '%s' is not a map", dllKey)
	}
	classesData, exists := dllMap["classes"]
	if !exists {
		log.Fatalf("Key 'classes' does not exist in '%s'", dllKey)
	}
	classesMap, ok := classesData.(map[string]interface{})
	if !ok {
		log.Fatalf("Value of 'classes' in '%s' is not a map", dllKey)
	}
	classData, exists := classesMap[classKey]
	if !exists {
		log.Fatalf("Class '%s' does not exist in 'classes' within '%s'", classKey, dllKey)
	}
	classMap, ok := classData.(map[string]interface{})
	if !ok {
		log.Fatalf("Value of class '%s' is not a map", classKey)
	}
	fieldsData, exists := classMap["fields"]
	if !exists {
		log.Fatalf("Key 'fields' does not exist in class '%s'", classKey)
	}
	fieldsMap, ok := fieldsData.(map[string]interface{})
	if !ok {
		log.Fatalf("Value of 'fields' in class '%s' is not a map", classKey)
	}
	fieldValue, exists := fieldsMap[fieldKey]
	if !exists {
		log.Fatalf("Field '%s' does not exist in 'fields' of class '%s'", fieldKey, classKey)
	}
	floatValue, ok := fieldValue.(float64)
	if !ok {
		log.Fatalf("Field '%s' in 'fields' of class '%s' is not of expected type (float64)", fieldKey, classKey)
	}
	return uintptr(floatValue)
}

func fetchAndCombineOffsets() {
	const offsetsURL = "https://raw.githubusercontent.com/a2x/cs2-dumper/refs/heads/main/output/offsets.json"
	const clientDllURL = "https://raw.githubusercontent.com/a2x/cs2-dumper/refs/heads/main/output/client_dll.json"
	offsetsResponse, err := http.Get(offsetsURL)
	if err != nil {
		log.Fatalf("Error downloading offsets.json: %v", err)
	}
	defer offsetsResponse.Body.Close()
	if offsetsResponse.StatusCode != http.StatusOK {
		log.Fatalf("Error downloading offsets.json - status code %d", offsetsResponse.StatusCode)
	}
	offsetsBody, err := ioutil.ReadAll(offsetsResponse.Body)
	if err != nil {
		log.Fatalf("Error reading the content of offsets.json: %v", err)
	}
	clientResponse, err := http.Get(clientDllURL)
	if err != nil {
		log.Fatalf("Error downloading client_dll.json: %v", err)
	}
	defer clientResponse.Body.Close()
	if clientResponse.StatusCode != http.StatusOK {
		log.Fatalf("Error downloading client_dll.json: status code %d", clientResponse.StatusCode)
	}
	clientBody, err := ioutil.ReadAll(clientResponse.Body)
	if err != nil {
		log.Fatalf("Error reading content of client_dll.json: %v", err)
	}
	var offsetsData map[string]interface{}
	var clientOffsetsData map[string]interface{}
	if err := json.Unmarshal(offsetsBody, &offsetsData); err != nil {
		log.Fatalf("Error decoding offsets.json: %v", err)
	}
	if err := json.Unmarshal(clientBody, &clientOffsetsData); err != nil {
		log.Fatalf("Error decoding client_dll.json: %v", err)
	}
	clientDllOffsets, ok := offsetsData["client.dll"].(map[string]interface{})
	if !ok {
		log.Fatalf("Error: 'client.dll' does not exist or is not of the expected type")
	}
	combinedOffsets := CombinedOffsets{
		DwEntityList:           getValue(clientDllOffsets, "dwEntityList"),
		DwLocalPlayerPawn:      getValue(clientDllOffsets, "dwLocalPlayerPawn"),
		DwViewMatrix:           getValue(clientDllOffsets, "dwViewMatrix"),
		M_hPlayerPawn:          getNestedFieldValue(clientOffsetsData, "client.dll", "CCSPlayerController", "m_hPlayerPawn"),
		M_sSanitizedPlayerName: getNestedFieldValue(clientOffsetsData, "client.dll", "CCSPlayerController", "m_sSanitizedPlayerName"),
		M_vOldOrigin:           getNestedFieldValue(clientOffsetsData, "client.dll", "C_BasePlayerPawn", "m_vOldOrigin"),
		M_modelState:           getNestedFieldValue(clientOffsetsData, "client.dll", "CSkeletonInstance", "m_modelState"),
		M_nodeToWorld:          getNestedFieldValue(clientOffsetsData, "client.dll", "CGameSceneNode", "m_nodeToWorld"),
		M_iHealth:              getNestedFieldValue(clientOffsetsData, "client.dll", "C_BaseEntity", "m_iHealth"),
		M_lifeState:            getNestedFieldValue(clientOffsetsData, "client.dll", "C_BaseEntity", "m_lifeState"),
		M_iTeamNum:             getNestedFieldValue(clientOffsetsData, "client.dll", "C_BaseEntity", "m_iTeamNum"),
		M_pGameSceneNode:       getNestedFieldValue(clientOffsetsData, "client.dll", "C_BaseEntity", "m_pGameSceneNode"),
		M_boneArray:            getNestedFieldValue(clientOffsetsData, "client.dll", "CSkeletonInstance", "m_modelState") + 0x80,
	}
	file, err := os.Create("offsets.json")
	if err != nil {
		log.Fatalf("Error creating offsets.json file: %v", err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(combinedOffsets); err != nil {
		log.Fatalf("Error writing offsets.json file: %v", err)
	}
	fmt.Println("Combined offsets.json file created successfully.")
}

type Matrix [4][4]float32
type Vector3 struct{ X, Y, Z float32 }

func (v Vector3) Dist(other Vector3) float32 {
	return float32(math.Abs(float64(v.X-other.X)) + math.Abs(float64(v.Y-other.Y)) + math.Abs(float64(v.Z-other.Z)))
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
type Offset struct {
	DwViewMatrix           uintptr `json:"dwViewMatrix"`
	DwLocalPlayerPawn      uintptr `json:"dwLocalPlayerPawn"`
	DwEntityList           uintptr `json:"dwEntityList"`
	M_hPlayerPawn          uintptr `json:"m_hPlayerPawn"`
	M_iHealth              uintptr `json:"m_iHealth"`
	M_lifeState            uintptr `json:"m_lifeState"`
	M_iTeamNum             uintptr `json:"m_iTeamNum"`
	M_vOldOrigin           uintptr `json:"m_vOldOrigin"`
	M_pGameSceneNode       uintptr `json:"m_pGameSceneNode"`
	M_modelState           uintptr `json:"m_modelState"`
	M_boneArray            uintptr `json:"m_boneArray"`
	M_nodeToWorld          uintptr `json:"m_nodeToWorld"`
	M_sSanitizedPlayerName uintptr `json:"m_sSanitizedPlayerName"`
}

var (
	user32                     = windows.NewLazySystemDLL("user32.dll")
	gdi32                      = windows.NewLazySystemDLL("gdi32.dll")
	getSystemMetrics           = user32.NewProc("GetSystemMetrics")
	setLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")
	showCursor                 = user32.NewProc("ShowCursor")
	fillRect                   = user32.NewProc("FillRect")
	setTextAlign               = gdi32.NewProc("SetTextAlign")
	createFont                 = gdi32.NewProc("CreateFontW")
	createCompatibleDC         = gdi32.NewProc("CreateCompatibleDC")
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

func worldToScreen(viewMatrix Matrix, position Vector3) (float32, float32) {
	screenX := viewMatrix[0][0]*position.X + viewMatrix[0][1]*position.Y + viewMatrix[0][2]*position.Z + viewMatrix[0][3]
	screenY := viewMatrix[1][0]*position.X + viewMatrix[1][1]*position.Y + viewMatrix[1][2]*position.Z + viewMatrix[1][3]
	w := viewMatrix[3][0]*position.X + viewMatrix[3][1]*position.Y + viewMatrix[3][2]*position.Z + viewMatrix[3][3]
	if w < 0.01 {
		return -1, -1
	}
	invw := 1.0 / w
	screenX *= invw
	screenY *= invw
	width, _, _ := getSystemMetrics.Call(0)
	height, _, _ := getSystemMetrics.Call(1)
	x, y := float32(width)/2, float32(height)/2
	x += 0.5*screenX*float32(width) + 0.5
	y -= 0.5*screenY*float32(height) + 0.5
	return x, y
}

func getOffsets() Offset {
	var offsets Offset
	offsetsJson, err := os.Open("offsets.json")
	if err != nil {
		fmt.Println("Error opening offsets.json", err)
		return offsets
	}
	defer offsetsJson.Close()
	if err := json.NewDecoder(offsetsJson).Decode(&offsets); err != nil {
		fmt.Println("Error decoding JSON:", err)
	}
	return offsets
}

func getEntitiesInfo(procHandle windows.Handle, clientDll, screenWidth, screenHeight uintptr, offsets Offset) []Entity {
	var entities []Entity
	var entityList uintptr
	if readSafe(procHandle, clientDll+offsets.DwEntityList, &entityList) != nil {
		return entities
	}
	var localPlayerP uintptr
	if readSafe(procHandle, clientDll+offsets.DwLocalPlayerPawn, &localPlayerP) != nil {
		return entities
	}
	if localPlayerP == 0 {
		return entities
	}
	var localPlayerGameScene uintptr
	if readSafe(procHandle, localPlayerP+offsets.M_pGameSceneNode, &localPlayerGameScene) != nil {
		return entities
	}
	var localPlayerSceneOrigin Vector3
	if readSafe(procHandle, localPlayerGameScene+offsets.M_nodeToWorld, &localPlayerSceneOrigin) != nil {
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
	bones := map[string]int{
		"head": 6, "neck_0": 5, "spine_1": 4, "spine_2": 2, "pelvis": 0, "arm_upper_L": 8, "arm_lower_L": 9, "hand_L": 10,
		"arm_upper_R": 13, "arm_lower_R": 14, "hand_R": 15, "leg_upper_L": 22, "leg_lower_L": 23, "ankle_L": 24,
		"leg_upper_R": 25, "leg_lower_R": 26, "ankle_R": 27,
	}
	for i := 0; i < 64; i++ {
		var listEntry, entityController, entityControllerPawn, entityPawn uintptr
		if readSafe(procHandle, entityList+uintptr((8*(i&0x7FFF)>>9)+16), &listEntry) != nil || listEntry == 0 {
			continue
		}
		if readSafe(procHandle, listEntry+uintptr(120)*uintptr(i&0x1FF), &entityController) != nil || entityController == 0 {
			continue
		}
		if readSafe(procHandle, entityController+offsets.M_hPlayerPawn, &entityControllerPawn) != nil || entityControllerPawn == 0 {
			continue
		}
		if readSafe(procHandle, entityList+uintptr(0x8*((entityControllerPawn&0x7FFF)>>9)+16), &listEntry) != nil || listEntry == 0 {
			continue
		}
		if readSafe(procHandle, listEntry+uintptr(120)*uintptr(entityControllerPawn&0x1FF), &entityPawn) != nil || entityPawn == 0 {
			continue
		}
		if entityPawn == localPlayerP {
			continue
		}
		var lifeState int32
		if readSafe(procHandle, entityPawn+offsets.M_lifeState, &lifeState) != nil || lifeState != 256 {
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
		var entityNameAddress uintptr
		if readSafe(procHandle, entityController+offsets.M_sSanitizedPlayerName, &entityNameAddress) != nil {
			continue
		}
		nameBytes := make([]byte, 128)
		if readSafe(procHandle, entityNameAddress, &nameBytes) != nil {
			continue
		}
		var sanitizedName strings.Builder
		for _, b := range nameBytes {
			if b == 0 {
				break
			}
			r := rune(b)
			if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsPunct(r) || unicode.IsSpace(r) {
				sanitizedName.WriteRune(r)
			}
		}
		sanitizedNameStr := sanitizedName.String()
		if sanitizedNameStr == "" {
			continue
		}
		var gameScene, entityBoneArray uintptr
		if readSafe(procHandle, entityPawn+offsets.M_pGameSceneNode, &gameScene) != nil || gameScene == 0 {
			continue
		}
		if readSafe(procHandle, gameScene+offsets.M_boneArray, &entityBoneArray) != nil || entityBoneArray == 0 {
			continue
		}
		var entityOrigin, entityHead Vector3
		entityBones := make(map[string]Vector2)
		for boneName, boneIndex := range bones {
			var currentBone Vector3
			if readSafe(procHandle, entityBoneArray+uintptr(boneIndex)*32, &currentBone) != nil {
				continue
			}
			if boneName == "head" {
				entityHead = currentBone
			}
			boneX, boneY := worldToScreen(viewMatrix, currentBone)
			entityBones[boneName] = Vector2{boneX, boneY}
		}
		if readSafe(procHandle, entityPawn+offsets.M_vOldOrigin, &entityOrigin) != nil {
			continue
		}
		entityHeadTop := Vector3{entityHead.X, entityHead.Y, entityHead.Z + 7}
		entityHeadBottom := Vector3{entityHead.X, entityHead.Y, entityHead.Z - 5}
		screenPosHeadX, screenPosHeadTopY := worldToScreen(viewMatrix, entityHeadTop)
		_, screenPosHeadBottomY := worldToScreen(viewMatrix, entityHeadBottom)
		screenPosFeetX, screenPosFeetY := worldToScreen(viewMatrix, entityOrigin)
		entityBoxTopVec := Vector3{entityOrigin.X, entityOrigin.Y, entityOrigin.Z + 70}
		_, screenPosBoxTop := worldToScreen(viewMatrix, entityBoxTopVec)
		if screenPosHeadX <= -1 || screenPosFeetY <= -1 || screenPosHeadX >= float32(screenWidth) || screenPosHeadTopY >= float32(screenHeight) {
			continue
		}
		boxHeight := screenPosFeetY - screenPosBoxTop
		tempEntity := Entity{
			Health:   health,
			Team:     teamNum,
			Name:     sanitizedNameStr,
			Distance: entityOrigin.Dist(localPlayerSceneOrigin),
			Position: Vector2{screenPosFeetX, screenPosFeetY},
			Bones:    entityBones,
			HeadPos:  Vector3{screenPosHeadX, screenPosHeadTopY, screenPosHeadBottomY},
			Rect:     Rectangle{screenPosBoxTop, screenPosFeetX - boxHeight/4, screenPosFeetX + boxHeight/4, screenPosFeetY},
		}
		entities = append(entities, tempEntity)
	}
	return entities
}

func drawSkeleton(hdc win.HDC, pen uintptr, bones map[string]Vector2) {
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
		text, _ := windows.UTF16PtrFromString(fmt.Sprintf("%d", hp))
		win.SetTextColor(hdc, win.RGB(0, 255, 50))
		setTextAlign.Call(uintptr(hdc), 0x00000002)
		if HealthBarRendering {
			win.TextOut(hdc, int32(rect.Left)-8, int32(int(rect.Bottom)+1-int(float64(int(rect.Bottom)+1-int(rect.Top))*float64(hp)/100.0)), text, int32(len(fmt.Sprintf("%d", hp))))
		} else {
			win.TextOut(hdc, int32(rect.Left)-4, int32(rect.Top), text, int32(len(fmt.Sprintf("%d", hp))))
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
		if !ok || (targetBonePos.X == 0 && targetBonePos.Y == 0) {
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
	fetchAndCombineOffsets()
	go RunGui()
	screenWidth, _, _ := getSystemMetrics.Call(0)
	screenHeight, _, _ := getSystemMetrics.Call(1)
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
	offsets := getOffsets()
	rect := &win.RECT{Left: 0, Top: 0, Right: int32(screenWidth), Bottom: int32(screenHeight)}

	frameDuration := time.Second / targetFPS

	for {
		frameStart := time.Now()

		var exitCode uint32
		if err := windows.GetExitCodeProcess(procHandle, &exitCode); err != nil || exitCode != STILL_ACTIVE {
			fmt.Println("Process cs2.exe not found or has exited. Exiting program.")
			break
		}
		memhdc := win.CreateCompatibleDC(hdc)
		memBitmap := win.CreateCompatibleBitmap(hdc, int32(screenWidth), int32(screenHeight))
		win.SelectObject(memhdc, win.HGDIOBJ(memBitmap))
		fillRect.Call(uintptr(memhdc), uintptr(unsafe.Pointer(rect)), bgBrush)
		win.SetBkMode(memhdc, win.TRANSPARENT)
		win.SelectObject(memhdc, win.HGDIOBJ(font))
		entities := getEntitiesInfo(procHandle, clientDll, screenWidth, screenHeight, offsets)

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
		win.DeleteObject(win.HGDIOBJ(memBitmap))
		win.DeleteDC(memhdc)

		elapsed := time.Since(frameStart)
		if elapsed < frameDuration {
			time.Sleep(frameDuration - elapsed)
		}
	}
}