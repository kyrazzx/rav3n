package main
import (
	"image/color"
	"math"
	"time"
	g "github.com/AllenDang/giu"
)
var (
	colBg         = color.RGBA{9, 9, 14, 255}
	colSidebar    = color.RGBA{12, 12, 18, 255}
	colSurface    = color.RGBA{18, 18, 26, 255}
	colSurfaceHi  = color.RGBA{24, 24, 34, 255}
	colBorder     = color.RGBA{38, 38, 54, 255}
	colAccent     = color.RGBA{139, 92, 246, 255}
	colAccentDim  = color.RGBA{109, 40, 217, 255}
	colAccentSoft = color.RGBA{139, 92, 246, 40}
	colText       = color.RGBA{232, 232, 240, 255}
	colTextMuted  = color.RGBA{110, 110, 132, 255}
	colTextDim    = color.RGBA{72, 72, 92, 255}
	colDanger     = color.RGBA{239, 68, 68, 255}
	colSuccess    = color.RGBA{52, 211, 153, 255}
	colTerrorist  = color.RGBA{255, 142, 120, 255}
	colCT         = color.RGBA{122, 120, 255, 255}
)
type themePreset struct {
	Name      string
	Accent    color.RGBA
	AccentDim color.RGBA
	AccentSoft color.RGBA
}
var themePresets = []themePreset{
	{Name: "Raven Purple", Accent: color.RGBA{139, 92, 246, 255}, AccentDim: color.RGBA{109, 40, 217, 255}, AccentSoft: color.RGBA{139, 92, 246, 40}},
	{Name: "Crimson Elite", Accent: color.RGBA{239, 68, 68, 255}, AccentDim: color.RGBA{185, 28, 28, 255}, AccentSoft: color.RGBA{239, 68, 68, 40}},
	{Name: "Ice Neon", Accent: color.RGBA{56, 189, 248, 255}, AccentDim: color.RGBA{14, 116, 144, 255}, AccentSoft: color.RGBA{56, 189, 248, 40}},
}
var selectedThemeIndex int32 = 0
type uiAnim struct {
	navIndicatorY    float32
	navIndicatorDest float32
	pageAlpha        float32
	pageSlide        float32
	pageSlideDir     float32
	lastSection      string
	pulse            float32
	started          time.Time
	lastDelta        float32
	lastFrameAt      time.Time
	toggleState      map[string]float32
	navHover         map[string]float32
}
var uiAnimation = uiAnim{
	navIndicatorY:    108,
	navIndicatorDest: 108,
	pageAlpha:        1,
	pageSlide:        1,
	started:          time.Now(),
	lastDelta:        0.016,
	toggleState:      map[string]float32{},
	navHover:         map[string]float32{},
}
func sectionIndex(section string) int {
	switch section {
	case sectionAimbot:
		return 0
	case sectionESP:
		return 1
	case sectionMisc:
		return 2
	case sectionConfig:
		return 3
	default:
		return 0
	}
}

func lerp(a, b, t float32) float32 {
	return a + (b-a)*t
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func easeOutCubic(t float32) float32 {
	t = clamp01(t)
	return 1 - float32(math.Pow(float64(1-t), 3))
}

func (a *uiAnim) tick() float32 {
	dt := float32(0.016)
	now := time.Now()
	if !a.lastFrameAt.IsZero() {
		dt = float32(now.Sub(a.lastFrameAt).Seconds())
		if dt <= 0 || dt > 0.1 {
			dt = 0.016
		}
	}
	a.lastFrameAt = now
	a.pulse = float32(math.Sin(float64(time.Since(a.started).Seconds())*2.2))*0.5 + 0.5
	a.navIndicatorY = lerp(a.navIndicatorY, a.navIndicatorDest, clamp01(dt*14))
	a.pageAlpha = lerp(a.pageAlpha, 1, clamp01(dt*10))
	a.pageSlide = lerp(a.pageSlide, 1, clamp01(dt*12))
	a.lastDelta = dt
	return dt
}

func (a *uiAnim) setSection(section string, destY float32) {
	if section != a.lastSection {
		if a.lastSection != "" {
			oldIdx := sectionIndex(a.lastSection)
			newIdx := sectionIndex(section)
			if newIdx >= oldIdx {
				a.pageSlideDir = 1
			} else {
				a.pageSlideDir = -1
			}
		}
		a.pageAlpha = 0
		a.pageSlide = 0
		a.lastSection = section
	}
	a.navIndicatorDest = destY
}

func (a *uiAnim) pageOffsetX() float32 {
	return (1 - easeOutCubic(a.pageSlide)) * a.pageSlideDir * 64
}

func (a *uiAnim) animateNavHover(id string, hovered bool) float32 {
	target := float32(0)
	if hovered {
		target = 1
	}
	cur := a.navHover[id]
	cur = lerp(cur, target, clamp01(a.lastDelta*16))
	a.navHover[id] = cur
	return cur
}

func (a *uiAnim) animateToggle(id string, enabled bool) float32 {
	target := float32(0)
	if enabled {
		target = 1
	}
	cur := a.toggleState[id]
	cur = lerp(cur, target, clamp01(a.lastDelta*18))
	a.toggleState[id] = cur
	return cur
}

func applyThemePreset(index int32) {
	if index < 0 || int(index) >= len(themePresets) {
		return
	}
	p := themePresets[index]
	colAccent = p.Accent
	colAccentDim = p.AccentDim
	colAccentSoft = p.AccentSoft
}

func applyRav3nTheme() *g.StyleSetter {
	return g.Style().
		SetColor(g.StyleColorText, colText).
		SetColor(g.StyleColorTextDisabled, colTextMuted).
		SetColor(g.StyleColorWindowBg, colBg).
		SetColor(g.StyleColorChildBg, colSurface).
		SetColor(g.StyleColorPopupBg, colSurfaceHi).
		SetColor(g.StyleColorBorder, colBorder).
		SetColor(g.StyleColorFrameBg, color.RGBA{28, 28, 40, 255}).
		SetColor(g.StyleColorFrameBgHovered, color.RGBA{34, 34, 48, 255}).
		SetColor(g.StyleColorFrameBgActive, color.RGBA{40, 40, 56, 255}).
		SetColor(g.StyleColorTitleBg, colSidebar).
		SetColor(g.StyleColorTitleBgActive, colSidebar).
		SetColor(g.StyleColorTitleBgCollapsed, colSidebar).
		SetColor(g.StyleColorCheckMark, colAccent).
		SetColor(g.StyleColorSliderGrab, colAccent).
		SetColor(g.StyleColorSliderGrabActive, colAccentDim).
		SetColor(g.StyleColorButton, color.RGBA{32, 32, 46, 255}).
		SetColor(g.StyleColorButtonHovered, color.RGBA{42, 42, 58, 255}).
		SetColor(g.StyleColorButtonActive, color.RGBA{50, 50, 68, 255}).
		SetColor(g.StyleColorHeader, color.RGBA{28, 28, 40, 180}).
		SetColor(g.StyleColorHeaderHovered, color.RGBA{36, 36, 52, 200}).
		SetColor(g.StyleColorHeaderActive, color.RGBA{44, 44, 62, 220}).
		SetColor(g.StyleColorSeparator, colBorder).
		SetColor(g.StyleColorScrollbarBg, colSidebar).
		SetColor(g.StyleColorScrollbarGrab, color.RGBA{50, 50, 68, 255}).
		SetColor(g.StyleColorScrollbarGrabHovered, colAccentSoft).
		SetColor(g.StyleColorScrollbarGrabActive, colAccent).
		SetStyleFloat(g.StyleVarWindowRounding, 14).
		SetStyleFloat(g.StyleVarChildRounding, 12).
		SetStyleFloat(g.StyleVarFrameRounding, 8).
		SetStyleFloat(g.StyleVarGrabRounding, 8).
		SetStyleFloat(g.StyleVarScrollbarRounding, 10).
		SetStyleFloat(g.StyleVarPopupRounding, 10).
		SetStyle(g.StyleVarWindowPadding, 0, 0).
		SetStyle(g.StyleVarFramePadding, 12, 8).
		SetStyle(g.StyleVarItemSpacing, 12, 10).
		SetStyleFloat(g.StyleVarIndentSpacing, 20)
}

func withAlpha(c color.RGBA, alpha float32) color.RGBA {
	return color.RGBA{c.R, c.G, c.B, uint8(float32(c.A) * clamp01(alpha))}
}
