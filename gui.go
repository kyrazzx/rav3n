package main

import (
	"image/color"
	"os"
	"sort"

	g "github.com/AllenDang/giu"
)

type recoilProfile struct {
	X, Y, Smooth float32
	StartBullet  int32
}

var (
	activeSideMenu        = "Aimbot"
	sidebarWidth          float32 = 200.0
	aimbotChildWidth      float32 = 240.0
	otherChildWidth       float32 = 240.0
	
	keyNames              []string
	keyMap                map[string]int32
	selectedKeyIndex      int32
	
	weaponNames           []string
	weaponConfigs         map[string]recoilProfile
	selectedWeaponIndex   int32

	aimbotTargetNames       []string
	aimbotTargetMap         map[string]string
	selectedAimbotTargetIndex int32
)

func init() {
	keyMap = map[string]int32{
		"Mouse 4":      0x05, // VK_XBUTTON1
		"Mouse 5":      0x06, // VK_XBUTTON2
		"Left Alt":     0xA4, // VK_LMENU
		"Left Shift":   0xA0, // VK_LSHIFT
		"Left Ctrl":    0xA2, // VK_LCONTROL
		"Caps Lock":    0x14, // VK_CAPITAL
	}
	keyNames = make([]string, 0, len(keyMap))
	for name := range keyMap {
		keyNames = append(keyNames, name)
	}
	sort.Strings(keyNames)
	for i, name := range keyNames {
		if keyMap[name] == AimbotKey {
			selectedKeyIndex = int32(i)
			break
		}
	}

	weaponConfigs = map[string]recoilProfile{
		"Default":      {X: 0.0, Y: 2.0, Smooth: 1.0, StartBullet: 1},
		"AK-47":        {X: 0.0, Y: 2.0, Smooth: 1.0, StartBullet: 2},
		"M4A4":         {X: 0.0, Y: 2.0, Smooth: 1.0, StartBullet: 2},
		"M4A1-S":       {X: 0.0, Y: 1.8, Smooth: 1.0, StartBullet: 2},
		"Galil AR":     {X: 0.0, Y: 1.7, Smooth: 1.0, StartBullet: 2},
		"FAMAS":        {X: 0.0, Y: 1.6, Smooth: 1.0, StartBullet: 2},
	}
	weaponNames = make([]string, 0, len(weaponConfigs))
	for name := range weaponConfigs {
		weaponNames = append(weaponNames, name)
	}
	sort.Strings(weaponNames)

	aimbotTargetMap = map[string]string{
		"Head":   "head",
		"Neck":   "neck_0",
		"Chest":  "spine_2",
		"Pelvis": "pelvis",
	}
	aimbotTargetNames = make([]string, 0, len(aimbotTargetMap))
	for name := range aimbotTargetMap {
		aimbotTargetNames = append(aimbotTargetNames, name)
	}
	sort.Strings(aimbotTargetNames)
	for i, name := range aimbotTargetNames {
		if aimbotTargetMap[name] == AimbotTarget {
			selectedAimbotTargetIndex = int32(i)
			break
		}
	}
}

func loop(wnd *g.MasterWindow) {
	g.SingleWindow().Layout(
		g.Custom(func() {
			dragBar := g.InvisibleButton().Size(g.Auto, 20)
			dragBar.Build()
			if g.IsItemActive() {
				delta := g.Context.IO().MouseDelta()
				x, y := wnd.GetPos()
				wnd.SetPos(x+int(delta.X), y+int(delta.Y))
			}
		}),
		g.Style().
			SetColor(g.StyleColorText, color.RGBA{R: 220, G: 220, B: 220, A: 255}).
			SetColor(g.StyleColorWindowBg, color.RGBA{R: 21, G: 21, B: 21, A: 255}).
			SetColor(g.StyleColorChildBg, color.RGBA{R: 28, G: 28, B: 28, A: 255}).
			SetColor(g.StyleColorBorder, color.RGBA{R: 40, G: 40, B: 40, A: 255}).
			SetColor(g.StyleColorFrameBg, color.RGBA{R: 45, G: 45, B: 45, A: 255}).
			SetColor(g.StyleColorFrameBgHovered, color.RGBA{R: 55, G: 55, B: 55, A: 255}).
			SetColor(g.StyleColorFrameBgActive, color.RGBA{R: 65, G: 65, B: 65, A: 255}).
			SetColor(g.StyleColorTitleBgActive, color.RGBA{R: 25, G: 25, B: 25, A: 255}).
			SetColor(g.StyleColorCheckMark, color.RGBA{R: 10, G: 200, B: 10, A: 255}).
			SetColor(g.StyleColorSliderGrab, color.RGBA{R: 10, G: 200, B: 10, A: 255}).
			SetColor(g.StyleColorSliderGrabActive, color.RGBA{R: 20, G: 220, B: 20, A: 255}).
			SetColor(g.StyleColorButton, color.RGBA{R: 50, G: 50, B: 50, A: 255}).
			SetColor(g.StyleColorButtonHovered, color.RGBA{R: 60, G: 60, B: 60, A: 255}).
			SetColor(g.StyleColorButtonActive, color.RGBA{R: 70, G: 70, B: 70, A: 255}).
			SetColor(g.StyleColorHeader, color.RGBA{R: 43, G: 43, B: 43, A: 255}).
			SetColor(g.StyleColorHeaderHovered, color.RGBA{R: 53, G: 53, B: 53, A: 255}).
			SetColor(g.StyleColorHeaderActive, color.RGBA{R: 63, G: 63, B: 63, A: 255}).
			SetColor(g.StyleColorSeparator, color.RGBA{R: 50, G: 50, B: 50, A: 255}).
			SetStyleFloat(g.StyleVarWindowRounding, 6.0).
			SetStyleFloat(g.StyleVarFrameRounding, 4.0).
			SetStyleFloat(g.StyleVarChildRounding, 4.0).
			SetStyleFloat(g.StyleVarGrabRounding, 2.0).
			To(
				g.SplitLayout(g.DirectionHorizontal, &sidebarWidth,
					buildSidebar(),
					buildMainContent(),
				),
			),
	)
}

func buildSidebar() g.Widget {
	return g.Child().Border(false).Layout(
		g.Label("RAV3N"),
		g.Separator(),
		g.Label("Combat"),
		g.Selectable("Aimbot").Selected(activeSideMenu == "Aimbot").OnClick(func() { activeSideMenu = "Aimbot" }),
		g.Label("Visuals"),
		g.Selectable("ESP").Selected(activeSideMenu == "ESP").OnClick(func() { activeSideMenu = "ESP" }),
		g.Label("Misc"),
		g.Selectable("Main").Selected(activeSideMenu == "Main").OnClick(func() { activeSideMenu = "Main" }),
		g.Button("Exit").OnClick(func() { os.Exit(0) }),
	)
}

func buildMainContent() g.Layout {
	return g.Layout{
		g.Row(
			g.Button("GLOBALS"),
		),
		g.Separator(),
		g.Custom(func() {
			switch activeSideMenu {
			case "Aimbot":
				buildAimbotPage()
			case "ESP":
				buildVisualsPage()
			case "Main":
				buildMiscPage()
			default:
				g.Label("Section not implemented").Build()
			}
		}),
	}
}

func buildAimbotPage() {
	g.Row(
		g.Child().Size(aimbotChildWidth, g.Auto).Border(true).Layout(
			g.Label("Aimbot"),
			g.Checkbox("Enable", &AimbotEnabled),
			g.SliderFloat(&AimbotFOV, 1.0, 500.0).Label("FOV"),
			g.SliderFloat(&AimbotSmoothing, 1.0, 50.0).Label("Smoothing"),
			g.Combo("Aimbot Key", keyNames[selectedKeyIndex], keyNames, &selectedKeyIndex).OnChange(func() {
				AimbotKey = keyMap[keyNames[selectedKeyIndex]]
			}),
			g.Combo("Aimbot Target", aimbotTargetNames[selectedAimbotTargetIndex], aimbotTargetNames, &selectedAimbotTargetIndex).OnChange(func() {
				AimbotTarget = aimbotTargetMap[aimbotTargetNames[selectedAimbotTargetIndex]]
			}),
		),
		g.Child().Size(otherChildWidth, g.Auto).Border(true).Layout(
			g.Label("Recoil"),
			g.Checkbox("Enable", &RecoilEnabled),
			g.Combo("Weapon Config", weaponNames[selectedWeaponIndex], weaponNames, &selectedWeaponIndex).OnChange(func() {
				selectedWeapon := weaponNames[selectedWeaponIndex]
				if config, ok := weaponConfigs[selectedWeapon]; ok {
					RecoilXAxis = config.X
					RecoilYAxis = config.Y
					RecoilSmooth = config.Smooth
					RecoilStartBullet = config.StartBullet
				}
			}),
			g.InputInt(&RecoilStartBullet).Label("Start bullet"),
			g.SliderFloat(&RecoilXAxis, -5.0, 5.0).Label("X Axis").Format("%.2f"),
			g.SliderFloat(&RecoilYAxis, 0.0, 5.0).Label("Y Axis").Format("%.2f"),
			g.SliderFloat(&RecoilSmooth, 1.0, 5.0).Label("Smooth").Format("%.2f"),
		),
	).Build()
}

func buildVisualsPage() {
	g.TreeNode("ESP").Flags(g.TreeNodeFlagsDefaultOpen).Layout(
		g.Checkbox("Enable Teammate Highlighting", &TeamCheck),
		g.Checkbox("Render ESP Boxes", &BoxRendering),
		g.Checkbox("Render Skeletons", &SkeletonRendering),
		g.Checkbox("Render Head Circles", &HeadCircle),
		g.Checkbox("Render Names", &NameRendering),
		g.Checkbox("Render Health Bar", &HealthBarRendering),
		g.Checkbox("Render Health Text", &HealthTextRendering),
	).Build()
}

func buildMiscPage() {
	g.TreeNode("Performance").Flags(g.TreeNodeFlagsDefaultOpen).Layout(
		g.Label("Higher delay = lower performance impact"),
		g.SliderInt(&FrameDelay, 1, 100).Label("Frame Delay (ms)"),
	).Build()
}

func RunGui() {
	wnd := g.NewMasterWindow("RAV3N v1.0.0", 850, 550, g.MasterWindowFlagsFrameless)
	wnd.Run(func() { loop(wnd) })
}