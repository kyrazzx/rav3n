package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strings"

	g "github.com/AllenDang/giu"
)

type recoilProfile struct {
	X, Y, Smooth float32
	StartBullet  int32
}

const (
	sectionAimbot = "aimbot"
	sectionESP    = "esp"
	sectionMisc   = "misc"
	sectionConfig = "config"
)

var (
	activeSection = sectionAimbot
	sidebarWidth  float32 = 200

	keyNames        []string
	keyMap          map[string]int32
	selectedKeyIndex int32

	weaponNames         []string
	weaponConfigs       map[string]recoilProfile
	selectedWeaponIndex int32

	aimbotTargetNames         []string
	aimbotTargetMap           map[string]string
	selectedAimbotTargetIndex int32

	configNameInput string
	exportNameInput string
	configStatus    = "No profile loaded"
	configFiles     []string
	configSelected  int32
)

type guiProfile struct {
	ThemeIndex           int32   `json:"themeIndex"`
	TeamCheck            bool    `json:"teamCheck"`
	HeadCircle           bool    `json:"headCircle"`
	SkeletonRendering    bool    `json:"skeletonRendering"`
	BoxRendering          bool    `json:"boxRendering"`
	BodyHighlightRendering bool   `json:"bodyHighlightRendering"`
	NameRendering         bool    `json:"nameRendering"`
	HealthBarRendering   bool    `json:"healthBarRendering"`
	HealthTextRendering  bool    `json:"healthTextRendering"`
	FrameDelay           int32   `json:"frameDelay"`
	AimbotEnabled        bool    `json:"aimbotEnabled"`
	AimbotFOV            float32 `json:"aimbotFov"`
	AimbotKey            int32   `json:"aimbotKey"`
	AimbotSmoothing      float32 `json:"aimbotSmoothing"`
	AimbotTarget         string  `json:"aimbotTarget"`
	RecoilEnabled        bool    `json:"recoilEnabled"`
	RecoilStartBullet    int32   `json:"recoilStartBullet"`
	RecoilXAxis          float32 `json:"recoilXAxis"`
	RecoilYAxis          float32 `json:"recoilYAxis"`
	RecoilSmooth         float32 `json:"recoilSmooth"`
	SelectedWeaponIndex  int32   `json:"selectedWeaponIndex"`
}

func profilesDir() string {
	return "configs"
}

func exportsDir() string {
	return filepath.Join(profilesDir(), "exports")
}

func sanitizeProfileName(name string) string {
	name = strings.TrimSpace(name)
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func exportProfilePath(name string) string {
	return filepath.Join(exportsDir(), sanitizeProfileName(name)+".json")
}

func listProfiles() []string {
	entries, err := os.ReadDir(profilesDir())
	if err != nil {
		return []string{}
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)
	return files
}

func snapshotProfile() guiProfile {
	return guiProfile{
		ThemeIndex:          selectedThemeIndex,
		TeamCheck:           TeamCheck,
		HeadCircle:          HeadCircle,
		SkeletonRendering:   SkeletonRendering,
		BoxRendering:          BoxRendering,
		BodyHighlightRendering: BodyHighlightRendering,
		NameRendering:         NameRendering,
		HealthBarRendering:  HealthBarRendering,
		HealthTextRendering: HealthTextRendering,
		FrameDelay:          FrameDelay,
		AimbotEnabled:       AimbotEnabled,
		AimbotFOV:           AimbotFOV,
		AimbotKey:           AimbotKey,
		AimbotSmoothing:     AimbotSmoothing,
		AimbotTarget:        AimbotTarget,
		RecoilEnabled:       RecoilEnabled,
		RecoilStartBullet:   RecoilStartBullet,
		RecoilXAxis:         RecoilXAxis,
		RecoilYAxis:         RecoilYAxis,
		RecoilSmooth:        RecoilSmooth,
		SelectedWeaponIndex: selectedWeaponIndex,
	}
}

func applyProfile(p guiProfile) {
	selectedThemeIndex = p.ThemeIndex
	applyThemePreset(selectedThemeIndex)
	TeamCheck = p.TeamCheck
	HeadCircle = p.HeadCircle
	SkeletonRendering = p.SkeletonRendering
	BoxRendering = p.BoxRendering
	BodyHighlightRendering = p.BodyHighlightRendering
	NameRendering = p.NameRendering
	HealthBarRendering = p.HealthBarRendering
	HealthTextRendering = p.HealthTextRendering
	FrameDelay = p.FrameDelay
	AimbotEnabled = p.AimbotEnabled
	AimbotFOV = p.AimbotFOV
	AimbotKey = p.AimbotKey
	AimbotSmoothing = p.AimbotSmoothing
	AimbotTarget = p.AimbotTarget
	RecoilEnabled = p.RecoilEnabled
	RecoilStartBullet = p.RecoilStartBullet
	RecoilXAxis = p.RecoilXAxis
	RecoilYAxis = p.RecoilYAxis
	RecoilSmooth = p.RecoilSmooth
	if p.SelectedWeaponIndex >= 0 && int(p.SelectedWeaponIndex) < len(weaponNames) {
		selectedWeaponIndex = p.SelectedWeaponIndex
	}
}

func saveProfileTo(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(snapshotProfile())
}

func loadProfileFrom(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	var p guiProfile
	if err := json.NewDecoder(file).Decode(&p); err != nil {
		return err
	}
	applyProfile(p)
	return nil
}

func init() {
	applyThemePreset(selectedThemeIndex)
	configNameInput = "default"
	exportNameInput = "my_profile"
	configFiles = listProfiles()
	keyMap = map[string]int32{
		"Mouse 4":    0x05,
		"Mouse 5":    0x06,
		"Left Alt":   0xA4,
		"Left Shift": 0xA0,
		"Left Ctrl":  0xA2,
		"Caps Lock":  0x14,
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
		"Default":  {X: 0.0, Y: 2.0, Smooth: 1.0, StartBullet: 1},
		"AK-47":    {X: 0.0, Y: 2.0, Smooth: 1.0, StartBullet: 2},
		"M4A4":     {X: 0.0, Y: 2.0, Smooth: 1.0, StartBullet: 2},
		"M4A1-S":   {X: 0.0, Y: 1.8, Smooth: 1.0, StartBullet: 2},
		"Galil AR": {X: 0.0, Y: 1.7, Smooth: 1.0, StartBullet: 2},
		"FAMAS":    {X: 0.0, Y: 1.6, Smooth: 1.0, StartBullet: 2},
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
	uiAnimation.tick()
	if uiAnimation.lastDelta > 0 {
		recordGuiFrame(1 / uiAnimation.lastDelta)
	}

	theme := applyRav3nTheme()
	theme.Push()
	g.Custom(func() {
		canvas := g.GetCanvas()
		pos := g.GetCursorScreenPos()
		w, h := g.GetAvailableRegion()
		canvas.AddRectFilled(pos, pos.Add(image.Pt(int(w), int(h))), colBg, 14, g.DrawFlagsRoundCornersAll)
		g.Dummy(w, h).Build()
	})
	g.SingleWindow().Layout(
		rav3nHeader(wnd),
		g.Child().Border(false).Size(-1, -1).Layout(
			g.Row(
				g.Child().ID("##rav3n_sidebar").Border(false).Size(sidebarWidth, -1).Layout(
					g.Style().SetStyle(g.StyleVarWindowPadding, 12, 14).To(buildSidebarContent()),
				),
				rav3nSidebarDivider(),
				g.Child().
					ID("##rav3n_content_scroll").
					Border(false).
					Size(-1, -1).
					Flags(g.WindowFlagsAlwaysVerticalScrollbar).
					Layout(
						g.Style().SetStyle(g.StyleVarWindowPadding, 20, 16).To(buildContentArea()),
					),
			),
		),
		rav3nStatusBar(),
	)
	theme.Pop()
}

func buildSidebarContent() g.Widget {
	return g.Layout{
		g.Dummy(0, 16),
		g.Style().SetColor(g.StyleColorText, colAccent).To(g.Label("R A V 3 N")),
		g.Style().SetColor(g.StyleColorText, colTextDim).To(g.Label("external suite")),
		g.Separator(),
		rav3nSectionLabel("COMBAT"),
		rav3nNavButton("Aimbot", sectionAimbot),
		g.Dummy(0, 4),
		rav3nSectionLabel("VISUALS"),
		rav3nNavButton("Player ESP", sectionESP),
		g.Dummy(0, 4),
		rav3nSectionLabel("MISC"),
		rav3nNavButton("Settings", sectionMisc),
		rav3nNavButton("Configs", sectionConfig),
		g.Dummy(-1, 12),
		g.Style().
			SetColor(g.StyleColorButton, color.RGBA{36, 24, 32, 255}).
			SetColor(g.StyleColorButtonHovered, color.RGBA{56, 28, 36, 255}).
			SetColor(g.StyleColorText, colDanger).
			To(g.Button("Exit Application").Size(-1, 34).OnClick(func() { os.Exit(0) })),
	}
}

func buildContentArea() g.Widget {
	return g.Layout{
		g.Dummy(0, 8),
		activePage(),
		g.Dummy(0, 48),
	}
}

func activePage() g.Widget {
	switch activeSection {
	case sectionAimbot:
		return buildAimbotPage()
	case sectionESP:
		return buildESPPage()
	case sectionMisc:
		return buildMiscPage()
	case sectionConfig:
		return buildConfigPage()
	default:
		return g.Layout{}
	}
}

func buildAimbotPage() g.Widget {
	return g.Layout{
		rav3nPageTitle("Combat", "Precision assistance & recoil control"),
		rav3nCard("Aimbot", "Target acquisition & smoothing", -1, 0, AimbotEnabled, g.Layout{
			rav3nToggle("aimbot_enabled", "Enabled", &AimbotEnabled),
			g.Dummy(0, 4),
			rav3nSliderFloat(&AimbotFOV, 1, 500, "Field of View", "%.0f"),
			rav3nSliderFloat(&AimbotSmoothing, 1, 30, "Smoothing", "%.1f"),
			rav3nCombo("Activation Key", keyNames[selectedKeyIndex], keyNames, &selectedKeyIndex, func() {
				AimbotKey = keyMap[keyNames[selectedKeyIndex]]
			}),
			rav3nCombo("Target Bone", aimbotTargetNames[selectedAimbotTargetIndex], aimbotTargetNames, &selectedAimbotTargetIndex, func() {
				AimbotTarget = aimbotTargetMap[aimbotTargetNames[selectedAimbotTargetIndex]]
			}),
		}),
		g.Dummy(0, 12),
		rav3nCard("Recoil Control", "Weapon-specific compensation", -1, 0, RecoilEnabled, g.Layout{
			rav3nToggle("recoil_enabled", "Enabled", &RecoilEnabled),
			g.Dummy(0, 4),
			rav3nCombo("Weapon Preset", weaponNames[selectedWeaponIndex], weaponNames, &selectedWeaponIndex, func() {
				selectedWeapon := weaponNames[selectedWeaponIndex]
				if config, ok := weaponConfigs[selectedWeapon]; ok {
					RecoilXAxis = config.X
					RecoilYAxis = config.Y
					RecoilSmooth = config.Smooth
					RecoilStartBullet = config.StartBullet
				}
			}),
			rav3nSliderInt(&RecoilStartBullet, 1, 10, "Start Bullet"),
			rav3nSliderFloat(&RecoilXAxis, -5, 5, "Compensate X", "%.2f"),
			rav3nSliderFloat(&RecoilYAxis, 0, 5, "Compensate Y", "%.2f"),
			rav3nSliderFloat(&RecoilSmooth, 1, 5, "Smooth Factor", "%.2f"),
		}),
		g.Dummy(0, 16),
	}
}

func buildESPPage() g.Widget {
	return g.Layout{
		rav3nPageTitle("Visuals", "Overlay elements & ESP preview"),
		rav3nCard("ESP Elements", "Toggle overlay components", -1, 0, true, g.Layout{
			rav3nToggle("esp_team_filter", "Team Filter", &TeamCheck),
			g.Dummy(0, 2),
			g.Style().SetColor(g.StyleColorText, colTextDim).To(
				g.Label("Hide teammates when enabled"),
			),
			g.Separator(),
			rav3nToggle("esp_box", "Bounding Boxes", &BoxRendering),
			rav3nToggle("esp_body", "Body Highlight", &BodyHighlightRendering),
			g.Style().SetColor(g.StyleColorText, colTextDim).To(
				g.Label("Filled silhouette along player bones"),
			),
			g.Dummy(0, 4),
			rav3nToggle("esp_skeleton", "Skeleton", &SkeletonRendering),
			rav3nToggle("esp_head", "Head Circle", &HeadCircle),
			rav3nToggle("esp_name", "Player Names", &NameRendering),
			rav3nToggle("esp_hpbar", "Health Bar", &HealthBarRendering),
			rav3nToggle("esp_hptext", "Health Text", &HealthTextRendering),
		}),
		g.Dummy(0, 12),
		rav3nCard("Live Preview", "Real-time ESP visualization", -1, 0, true, g.Layout{
			g.Child().ID("##esp_preview_panel").Border(false).Size(-1, 340).Flags(g.WindowFlagsNoScrollbar|g.WindowFlagsNoScrollWithMouse).Layout(
				rav3nESPPreview(),
			),
		}),
		g.Dummy(0, 16),
	}
}

func buildMiscPage() g.Widget {
	return g.Layout{
		rav3nPageTitle("Settings", "Performance & application info"),
		rav3nCard("Performance", "Frame pacing & resource usage", -1, 0, true, g.Layout{
			g.Style().SetColor(g.StyleColorText, colTextMuted).To(
				g.Label("Higher values reduce CPU usage"),
			),
			g.Dummy(0, 4),
			rav3nSliderInt(&FrameDelay, 1, 16, "Min Frame Time (ms)"),
			g.Dummy(0, 8),
			rav3nCombo("Theme Preset", themePresets[selectedThemeIndex].Name, []string{
				themePresets[0].Name, themePresets[1].Name, themePresets[2].Name,
			}, &selectedThemeIndex, func() { applyThemePreset(selectedThemeIndex) }),
			g.Dummy(0, 10),
			g.Child().ID("##perf_graph_panel").Border(false).Size(-1, 140).Flags(g.WindowFlagsNoScrollbar).Layout(
				rav3nPerfGraph(),
			),
		}),
		g.Dummy(0, 12),
		rav3nCard("About", "RAV3N external overlay", -1, 0, false, g.Layout{
			g.Style().SetColor(g.StyleColorText, colText).To(g.Label("RAV3N v2.1")),
			g.Style().SetColor(g.StyleColorText, colTextMuted).To(
				g.Label("Counter-Strike 2 external overlay"),
			),
			g.Dummy(0, 8),
			g.Separator(),
			g.Style().SetColor(g.StyleColorText, colTextDim).To(
				g.Label("Offsets: a2x/cs2-dumper"),
			),
			g.Style().SetColor(g.StyleColorText, colTextDim).To(
				g.Label("Auto-refresh on startup"),
			),
		}),
		g.Dummy(0, 16),
	}
}

func buildConfigPage() g.Widget {
	if len(configFiles) == 0 {
		configFiles = []string{"<none>"}
		configSelected = 0
	} else if int(configSelected) >= len(configFiles) {
		configSelected = int32(len(configFiles) - 1)
	}
	return g.Layout{
		rav3nPageTitle("Configs", "Save, load and share premium profiles"),
		rav3nCard("Profile Manager", "Persistent JSON configs", -1, 0, true, g.Layout{
			g.Label("Profile Name"),
			g.InputText(&configNameInput).Size(-1),
			g.Dummy(0, 6),
			g.Row(
				g.Button("Save").Size(100, 30).OnClick(func() {
					name := sanitizeProfileName(configNameInput)
					if name == "" {
						configStatus = "Profile name required"
						return
					}
					path := filepath.Join(profilesDir(), name+".json")
					if err := saveProfileTo(path); err != nil {
						configStatus = "Save failed: " + err.Error()
						return
					}
					configStatus = "Saved " + name
					configFiles = listProfiles()
				}),
				g.Button("Load").Size(100, 30).OnClick(func() {
					if len(configFiles) == 0 || configFiles[0] == "<none>" {
						configStatus = "No profile available"
						return
					}
					path := filepath.Join(profilesDir(), configFiles[configSelected])
					if err := loadProfileFrom(path); err != nil {
						configStatus = "Load failed: " + err.Error()
						return
					}
					configStatus = "Loaded " + configFiles[configSelected]
				}),
				g.Button("Delete").Size(100, 30).OnClick(func() {
					if len(configFiles) == 0 || configFiles[0] == "<none>" {
						configStatus = "No profile available"
						return
					}
					path := filepath.Join(profilesDir(), configFiles[configSelected])
					if err := os.Remove(path); err != nil {
						configStatus = "Delete failed: " + err.Error()
						return
					}
					configStatus = "Deleted " + configFiles[configSelected]
					configFiles = listProfiles()
				}),
			),
			g.Dummy(0, 8),
			g.Label("Available Profiles"),
			g.Combo("##profiles", configFiles[configSelected], configFiles, &configSelected).Size(-1),
		}),
		g.Dummy(0, 12),
		rav3nCard("Share & Theme", "Export/import + UI customization", -1, 0, true, g.Layout{
			g.Label("Export Name"),
			g.InputText(&exportNameInput).Size(-1),
			g.Dummy(0, 6),
			g.Row(
				g.Button("Export").Size(150, 30).OnClick(func() {
					name := sanitizeProfileName(exportNameInput)
					if name == "" {
						configStatus = "Export name required"
						return
					}
					path := exportProfilePath(name)
					if err := saveProfileTo(path); err != nil {
						configStatus = "Export failed: " + err.Error()
						return
					}
					configStatus = "Exported to " + path
				}),
				g.Button("Import").Size(150, 30).OnClick(func() {
					name := sanitizeProfileName(exportNameInput)
					if name == "" {
						configStatus = "Import name required"
						return
					}
					path := exportProfilePath(name)
					if err := loadProfileFrom(path); err != nil {
						configStatus = "Import failed: " + err.Error()
						return
					}
					configStatus = "Imported " + path
				}),
			),
			g.Dummy(0, 10),
			rav3nCombo("Theme Preset", themePresets[selectedThemeIndex].Name, []string{
				themePresets[0].Name, themePresets[1].Name, themePresets[2].Name,
			}, &selectedThemeIndex, func() { applyThemePreset(selectedThemeIndex) }),
			g.Dummy(0, 6),
			g.Style().SetColor(g.StyleColorText, colTextMuted).To(g.Label(fmt.Sprintf("Status: %s", configStatus))),
		}),
		g.Dummy(0, 16),
	}
}

func RunGui() {
	wnd := g.NewMasterWindow("RAV3N", 1100, 780, g.MasterWindowFlagsFrameless)
	wnd.Run(func() { loop(wnd) })
}
