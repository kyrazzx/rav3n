package main

import (
	"fmt"
	"image"
	"image/color"
	"os"

	g "github.com/AllenDang/giu"
)

func rav3nHeader(wnd *g.MasterWindow) g.Widget {
	return g.Custom(func() {
		canvas := g.GetCanvas()
		pos := g.GetCursorScreenPos()
		w := g.GetContentRegionAvail().X
		h := float32(52)

		canvas.AddRectFilled(pos, pos.Add(image.Pt(int(w), int(h))), colSurfaceHi, 0, 0)
		canvas.AddLine(pos.Add(image.Pt(0, int(h))), pos.Add(image.Pt(int(w), int(h))), colBorder, 1)

		shimmerW := float32(120)
		shimmerX := pos.X + uiAnimation.pulse*(w-shimmerW)
		canvas.AddRectFilled(
			image.Pt(int(shimmerX), pos.Y+int(h)-2),
			image.Pt(int(shimmerX+shimmerW), pos.Y+int(h)),
			withAlpha(colAccent, 0.35+uiAnimation.pulse*0.25), 0, 0,
		)

		g.SetCursorPos(image.Pt(20, 16))
		g.Label("RAV3N").Build()
		g.SameLine()
		g.SetCursorPosX(g.GetCursorPosX() + 8)
		g.Style().SetColor(g.StyleColorText, colTextMuted).To(g.Label("v2.1")).Build()

		g.SetCursorPos(image.Pt(int(w)-130, 14))
		g.Style().
			SetColor(g.StyleColorButton, color.RGBA{32, 32, 46, 255}).
			SetColor(g.StyleColorButtonHovered, color.RGBA{50, 50, 68, 255}).
			To(g.Button("•").Size(28, 28)).Build()

		g.SameLine()
		g.Style().
			SetColor(g.StyleColorButton, color.RGBA{46, 28, 32, 255}).
			SetColor(g.StyleColorButtonHovered, color.RGBA{72, 36, 40, 255}).
			SetColor(g.StyleColorText, colDanger).
			To(g.Button("×").Size(28, 28).OnClick(func() { os.Exit(0) })).Build()

		g.SetCursorPos(image.Pt(0, 0))
		drag := g.InvisibleButton().ID("##title_drag").Size(g.Auto, h)
		drag.Build()
		if g.IsItemActive() {
			delta := g.Context.IO().MouseDelta()
			x, y := wnd.GetPos()
			wnd.SetPos(x+int(delta.X), y+int(delta.Y))
		}

		g.SetCursorPos(image.Pt(0, int(h)))
	})
}

func rav3nSectionLabel(text string) g.Widget {
	return g.Custom(func() {
		g.Dummy(0, 6).Build()
		g.Style().SetColor(g.StyleColorText, colTextDim).To(
			g.Label(text),
		).Build()
		g.Dummy(0, 2).Build()
	})
}

func rav3nNavItem(id, label, icon string, section string, destY float32) g.Widget {
	return g.Custom(func() {
		active := activeSection == section
		pos := g.GetCursorScreenPos()
		w := g.GetContentRegionAvail().X
		h := float32(38)
		canvas := g.GetCanvas()

		g.SetCursorPos(pos)
		btn := g.InvisibleButton().ID("##nav_" + id).Size(w, h)
		btn.Build()
		hovered := g.IsItemHovered()
		hoverAnim := easeOutCubic(uiAnimation.animateNavHover(id, hovered))

		if active {
			bg := withAlpha(colAccent, 0.12+hoverAnim*0.06)
			canvas.AddRectFilled(pos, pos.Add(image.Pt(int(w), int(h))), bg, 6, g.DrawFlagsRoundCornersRight)
		} else if hoverAnim > 0.01 {
			bg := withAlpha(colSurfaceHi, 0.35+hoverAnim*0.35)
			canvas.AddRectFilled(pos, pos.Add(image.Pt(int(w), int(h))), bg, 6, g.DrawFlagsRoundCornersRight)
			canvas.AddRectFilled(
				image.Pt(int(pos.X), int(pos.Y+6)),
				image.Pt(int(pos.X)+2, int(pos.Y+h-6)),
				withAlpha(colAccent, 0.2+hoverAnim*0.5), 2, g.DrawFlagsRoundCornersRight,
			)
		}

		iconOffset := int(hoverAnim * 3)
		g.SetCursorPos(image.Pt(int(pos.X)+14+iconOffset, int(pos.Y)+10))
		g.Style().SetColor(g.StyleColorText, func() color.RGBA {
			if active {
				return colAccent
			}
			if hoverAnim > 0 {
				return color.RGBA{
					R: uint8(lerp(float32(colTextMuted.R), float32(colAccent.R), hoverAnim)),
					G: uint8(lerp(float32(colTextMuted.G), float32(colAccent.G), hoverAnim)),
					B: uint8(lerp(float32(colTextMuted.B), float32(colAccent.B), hoverAnim)),
					A: 255,
				}
			}
			return colTextMuted
		}()).To(g.Label(icon)).Build()

		g.SameLine()
		g.SetCursorPosX(g.GetCursorPosX() + 4)
		g.Style().SetColor(g.StyleColorText, func() color.RGBA {
			if active {
				return colText
			}
			if hoverAnim > 0 {
				return color.RGBA{
					R: uint8(lerp(float32(colTextMuted.R), float32(colText.R), hoverAnim)),
					G: uint8(lerp(float32(colTextMuted.G), float32(colText.G), hoverAnim)),
					B: uint8(lerp(float32(colTextMuted.B), float32(colText.B), hoverAnim)),
					A: 255,
				}
			}
			return colTextMuted
		}()).To(g.Label(label)).Build()

		if g.IsItemClicked() {
			activeSection = section
			uiAnimation.setSection(section, destY)
			if section == sectionConfig {
				configFiles = listProfiles()
			}
		}
		g.SetCursorPos(image.Pt(int(pos.X), int(pos.Y+h)))
	})
}

func rav3nSidebarIndicator() g.Widget {
	return g.Custom(func() {
		canvas := g.GetCanvas()
		x := float32(g.GetCursorScreenPos().X)
		y := uiAnimation.navIndicatorY
		canvas.AddRectFilled(
			image.Pt(int(x), int(y)),
			image.Pt(int(x)+3, int(y)+28),
			colAccent, 2, g.DrawFlagsRoundCornersRight,
		)
	})
}

func rav3nCard(title, subtitle string, width, height float32, glow bool, content g.Layout) g.Widget {
	return g.Child().Border(true).Size(width, height).Layout(
		g.Custom(func() {
			canvas := g.GetCanvas()
			pos := g.GetCursorScreenPos()
			w := g.GetContentRegionAvail().X
			if glow {
				glowCol := withAlpha(colAccent, 0.05+uiAnimation.pulse*0.12)
				canvas.AddRectFilled(
					image.Pt(pos.X-1, pos.Y-1),
					image.Pt(pos.X+int(w)+1, pos.Y+4),
					glowCol, 0, 0,
				)
			}
			canvas.AddRectFilled(pos, pos.Add(image.Pt(int(w), 4)), colAccent, 0, 0)
		}),
		g.Dummy(0, 8),
		g.Row(
			g.Custom(func() {
				g.Style().SetColor(g.StyleColorText, colText).To(g.Label(title)).Build()
			}),
		),
		g.Style().SetColor(g.StyleColorText, colTextMuted).To(
			g.Label(subtitle),
		),
		g.Separator(),
		g.Dummy(0, 4),
		content,
	)
}

func rav3nToggle(id, label string, value *bool) g.Widget {
	return g.Custom(func() {
		g.Style().SetColor(g.StyleColorText, colText).To(g.Label(label)).Build()
		g.SameLine()
		avail := g.GetContentRegionAvail().X
		g.SetCursorPosX(g.GetCursorPosX() + avail - 44)

		pos := g.GetCursorScreenPos()
		canvas := g.GetCanvas()
		trackW, trackH := float32(40), float32(20)
		knobR := float32(7)
		progress := easeOutCubic(uiAnimation.animateToggle(id, *value))

		trackColor := color.RGBA{36, 36, 50, 255}
		trackColor = color.RGBA{
			R: uint8(lerp(float32(trackColor.R), float32(colAccent.R), progress)),
			G: uint8(lerp(float32(trackColor.G), float32(colAccent.G), progress)),
			B: uint8(lerp(float32(trackColor.B), float32(colAccent.B), progress)),
			A: 255,
		}
		canvas.AddRectFilled(pos, pos.Add(image.Pt(int(trackW), int(trackH))), trackColor, int(trackH/2), g.DrawFlagsRoundCornersAll)

		knobX := pos.X + 10 + (trackW-20)*progress
		canvas.AddCircleFilled(image.Pt(int(knobX), pos.Y+int(trackH/2)), int(knobR), colText, 16)

		g.SetCursorPos(pos)
		btn := g.InvisibleButton().ID("##toggle_" + id).Size(trackW, trackH)
		btn.Build()
		if g.IsItemClicked() {
			*value = !*value
		}
		g.Dummy(trackW, trackH).Build()
	})
}

func rav3nSliderFloat(value *float32, min, max float32, label, format string) g.Widget {
	return g.Layout{
		g.Row(
			g.Style().SetColor(g.StyleColorText, colText).To(g.Label(label)),
			g.Dummy(-1, 0),
			g.Style().SetColor(g.StyleColorText, colAccent).To(
				g.Label(fmt.Sprintf(format, *value)),
			),
		),
		g.SliderFloat(value, min, max).Size(-1),
	}
}

func rav3nSliderInt(value *int32, min, max int32, label string) g.Widget {
	return g.Layout{
		g.Row(
			g.Style().SetColor(g.StyleColorText, colText).To(g.Label(label)),
			g.Dummy(-1, 0),
			g.Style().SetColor(g.StyleColorText, colAccent).To(
				g.Label(fmt.Sprintf("%d", *value)),
			),
		),
		g.SliderInt(value, min, max).Size(-1),
	}
}

func rav3nCombo(label string, preview string, items []string, selected *int32, onChange func()) g.Widget {
	return g.Custom(func() {
		g.Style().SetColor(g.StyleColorText, colText).To(g.Label(label)).Build()
		g.Combo(label+"##combo", preview, items, selected).Size(-1).OnChange(onChange).Build()
	})
}

func rav3nESPPreview() g.Widget {
	return g.Custom(func() {
		pos := g.GetCursorScreenPos()
		w := g.GetContentRegionAvail().X
		h := float32(340)
		canvas := g.GetCanvas()

		canvas.AddRectFilled(pos, pos.Add(image.Pt(int(w), int(h))), color.RGBA{14, 14, 20, 255}, 8, g.DrawFlagsRoundCornersAll)
		canvas.AddRect(pos, pos.Add(image.Pt(int(w), int(h))), colBorder, 8, g.DrawFlagsRoundCornersAll, 1)

		cx := pos.X + w/2
		top := pos.Y + 40
		bottom := pos.Y + h - 30
		boxW := float32(70)

		if BoxRendering {
			boxCol := colTerrorist
			if TeamCheck {
				boxCol = colCT
			}
			canvas.AddRect(
				image.Pt(int(cx-boxW/2), int(top)),
				image.Pt(int(cx+boxW/2), int(bottom)),
				boxCol, 0, 0, 2,
			)
		}

		if HealthBarRendering {
			barX := int(cx - boxW/2 - 6)
			canvas.AddRectFilled(
				image.Pt(barX, int(bottom)),
				image.Pt(barX+3, int(top+(bottom-top)*0.3)),
				colSuccess, 0, 0,
			)
		}

		if SkeletonRendering {
			skCol := withAlpha(colText, 0.7)
			headY := top + 18
			neckY := top + 38
			chestY := top + 80
			pelvisY := top + 130
			canvas.AddLine(image.Pt(int(cx), int(headY)), image.Pt(int(cx), int(pelvisY)), skCol, 1.5)
			canvas.AddLine(image.Pt(int(cx), int(neckY)), image.Pt(int(cx-28), int(chestY+20)), skCol, 1.5)
			canvas.AddLine(image.Pt(int(cx), int(neckY)), image.Pt(int(cx+28), int(chestY+20)), skCol, 1.5)
			canvas.AddLine(image.Pt(int(cx), int(pelvisY)), image.Pt(int(cx-18), int(bottom-10)), skCol, 1.5)
			canvas.AddLine(image.Pt(int(cx), int(pelvisY)), image.Pt(int(cx+18), int(bottom-10)), skCol, 1.5)
		}

		if HeadCircle {
			canvas.AddCircle(image.Pt(int(cx), int(top+18)), 14, colAccent, 16, 1.5)
		}

		if NameRendering {
			g.SetCursorPos(image.Pt(int(cx-30), int(top-18)))
			g.Style().SetColor(g.StyleColorText, colText).To(g.Label("Player")).Build()
		}

		g.Dummy(w, h).Build()
	})
}

func rav3nStatusBar() g.Widget {
	return g.Custom(func() {
		canvas := g.GetCanvas()
		pos := g.GetCursorScreenPos()
		w := g.GetContentRegionAvail().X
		h := float32(28)
		canvas.AddRectFilled(pos, pos.Add(image.Pt(int(w), int(h))), colSidebar, 0, 0)
		canvas.AddLine(pos, pos.Add(image.Pt(int(w), 0)), colBorder, 1)

		dotCol := colSuccess
		canvas.AddCircleFilled(pos.Add(image.Pt(14, 14)), 4, dotCol, 8)

		g.SetCursorPos(image.Pt(26, 6))
		g.Style().SetColor(g.StyleColorText, colTextMuted).To(
			g.Label("Ready  ·  CS2 External  ·  Offsets via cs2-dumper"),
		).Build()

		g.SetCursorPos(image.Pt(int(w)-120, 6))
		g.Style().SetColor(g.StyleColorText, colTextDim).To(g.Label("RAV3N v2.1")).Build()
		g.SetCursorPos(image.Pt(0, int(h)))
	})
}

func rav3nPageTitle(title, subtitle string) g.Widget {
	return g.Custom(func() {
		g.Dummy(0, 8).Build()
		g.Style().SetColor(g.StyleColorText, colText).To(
			g.Label(title),
		).Build()
		g.Style().SetColor(g.StyleColorText, colTextMuted).To(
			g.Label(subtitle),
		).Build()
		g.Dummy(0, 12).Build()
	})
}

func rav3nPerfGraph() g.Widget {
	return g.Custom(func() {
		pos := g.GetCursorScreenPos()
		w := g.GetContentRegionAvail().X
		h := float32(140)
		canvas := g.GetCanvas()

		canvas.AddRectFilled(pos, pos.Add(image.Pt(int(w), int(h))), color.RGBA{14, 14, 20, 255}, 8, g.DrawFlagsRoundCornersAll)
		canvas.AddRect(pos, pos.Add(image.Pt(int(w), int(h))), colBorder, 8, g.DrawFlagsRoundCornersAll, 1)

		pad := float32(12)
		graphLeft := pos.X + pad
		graphRight := pos.X + w - pad
		graphTop := pos.Y + pad + 8
		graphBottom := pos.Y + h - pad
		graphW := graphRight - graphLeft
		graphH := graphBottom - graphTop

		canvas.AddLine(image.Pt(int(graphLeft), int(graphBottom)), image.Pt(int(graphRight), int(graphBottom)), colBorder, 1)
		canvas.AddLine(image.Pt(int(graphLeft), int(graphTop)), image.Pt(int(graphLeft), int(graphBottom)), colBorder, 1)

		targetFPS := float32(120)
		for i := 0; i <= 4; i++ {
			y := graphBottom - graphH*float32(i)/4
			canvas.AddLine(image.Pt(int(graphLeft), int(y)), image.Pt(int(graphRight), int(y)), withAlpha(colBorder, 0.5), 1)
		}

		drawSeries := func(samples []float32, maxVal float32, col color.RGBA) {
			if len(samples) < 2 {
				return
			}
			step := graphW / float32(len(samples)-1)
			for i := 1; i < len(samples); i++ {
				v0 := clamp01(samples[i-1] / maxVal)
				v1 := clamp01(samples[i] / maxVal)
				x0 := graphLeft + step*float32(i-1)
				x1 := graphLeft + step*float32(i)
				y0 := graphBottom - v0*graphH
				y1 := graphBottom - v1*graphH
				canvas.AddLine(image.Pt(int(x0), int(y0)), image.Pt(int(x1), int(y1)), col, 2)
			}
		}

		drawSeries(perfGuiHistory, targetFPS, colAccent)
		overlayAsFPS := make([]float32, len(perfOverlayHistory))
		for i, ms := range perfOverlayHistory {
			if ms > 0 {
				overlayAsFPS[i] = 1000 / ms
			}
		}
		drawSeries(overlayAsFPS, targetFPS, colSuccess)

		g.SetCursorPos(image.Pt(int(pos.X)+12, int(pos.Y)+8))
		g.Style().SetColor(g.StyleColorText, colTextMuted).To(g.Label("Performance Monitor")).Build()
		g.SetCursorPos(image.Pt(int(pos.X)+12, int(pos.Y)+int(h)-22))
		g.Style().SetColor(g.StyleColorText, colAccent).To(
			g.Label(fmt.Sprintf("GUI %.0f FPS", perfGuiFPS)),
		).Build()
		g.SameLine()
		g.Style().SetColor(g.StyleColorText, colSuccess).To(
			g.Label(fmt.Sprintf("  Overlay %.1f ms", perfOverlayMs)),
		).Build()

		g.Dummy(w, h).Build()
	})
}
