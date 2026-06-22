package main

import (
	"fmt"
	"image"
	"image/color"
	"os"

	g "github.com/AllenDang/giu"
)

func rav3nSidebarDivider() g.Widget {
	return g.Custom(func() {
		pos := g.GetCursorScreenPos()
		_, h := g.GetAvailableRegion()
		canvas := g.GetCanvas()
		canvas.AddRectFilled(pos, pos.Add(image.Pt(1, int(h))), colBorder, 0, 0)
		g.Dummy(1, h).Build()
	})
}

func rav3nHeader(wnd *g.MasterWindow) g.Widget {
	return g.Custom(func() {
		canvas := g.GetCanvas()
		pos := g.GetCursorScreenPos()
		w, _ := g.GetAvailableRegion()
		h := float32(52)

		canvas.AddRectFilled(pos, pos.Add(image.Pt(int(w), int(h))), colSurfaceHi, 0, 0)
		canvas.AddLine(pos.Add(image.Pt(0, int(h))), pos.Add(image.Pt(int(w), int(h))), colBorder, 1)

		g.SetCursorPos(image.Pt(20, 16))
		g.Style().SetColor(g.StyleColorText, colAccent).To(g.Label("RAV3N")).Build()
		g.SameLine()
		cursorPos := g.GetCursorPos()
		g.SetCursorPos(image.Pt(cursorPos.X+8, cursorPos.Y+2))
		g.Style().SetColor(g.StyleColorText, colTextMuted).To(g.Label("v2.1")).Build()

		g.SetCursorPos(image.Pt(int(w)-96, 12))
		g.Style().
			SetColor(g.StyleColorButton, color.RGBA{46, 28, 32, 255}).
			SetColor(g.StyleColorButtonHovered, color.RGBA{72, 36, 40, 255}).
			SetColor(g.StyleColorText, colDanger).
			To(g.Button("×").Size(32, 32).OnClick(func() { os.Exit(0) })).Build()

		g.SetCursorPos(image.Pt(0, 0))
		g.InvisibleButton().ID("##title_drag").Size(-1, h).Build()
		if g.IsItemActive() {
			delta := g.Context.IO().GetMouseDelta()
			x, y := wnd.GetPos()
			wnd.SetPos(x+int(delta.X), y+int(delta.Y))
		}

		g.Dummy(-1, h).Build()
	})
}

func rav3nSectionLabel(text string) g.Widget {
	return g.Style().SetColor(g.StyleColorText, colTextDim).To(g.Label(text))
}

func rav3nNavButton(label, section string) g.Widget {
	active := activeSection == section
	widget := g.Button(label).Size(-1, 36).OnClick(func() {
		activeSection = section
		uiAnimation.setSection(section, 0)
		if section == sectionConfig {
			configFiles = listProfiles()
		}
	})
	if active {
		return g.Style().
			SetColor(g.StyleColorButton, color.RGBA{48, 32, 72, 255}).
			SetColor(g.StyleColorButtonHovered, color.RGBA{58, 38, 88, 255}).
			SetColor(g.StyleColorText, colAccent).
			To(widget)
	}
	return g.Style().
		SetColor(g.StyleColorButton, color.RGBA{22, 22, 32, 255}).
		SetColor(g.StyleColorButtonHovered, color.RGBA{32, 32, 46, 255}).
		SetColor(g.StyleColorText, colTextMuted).
		To(widget)
}

func rav3nCard(title, subtitle string, _width, _height float32, glow bool, content g.Layout) g.Widget {
	borderCol := colBorder
	bgCol := color.RGBA{20, 20, 30, 255}
	if glow {
		borderCol = withAlpha(colAccent, 0.45)
		bgCol = color.RGBA{24, 20, 36, 255}
	}
	return g.Layout{
		g.Custom(func() {
			pos := g.GetCursorScreenPos()
			w, _ := g.GetAvailableRegion()
			canvas := g.GetCanvas()
			canvas.AddRectFilled(pos, pos.Add(image.Pt(int(w), 8)), bgCol, 10, g.DrawFlagsRoundCornersTop)
			if glow {
				canvas.AddRectFilled(pos, pos.Add(image.Pt(int(w), 3)), colAccent, 10, g.DrawFlagsRoundCornersTop)
			}
			g.Dummy(w, 8).Build()
		}),
		g.Dummy(0, 10),
		g.Style().SetColor(g.StyleColorText, colText).To(g.Label(title)),
		g.Style().SetColor(g.StyleColorText, colTextMuted).To(g.Label(subtitle)),
		g.Custom(func() {
			pos := g.GetCursorScreenPos()
			w, _ := g.GetAvailableRegion()
			canvas := g.GetCanvas()
			canvas.AddLine(pos, pos.Add(image.Pt(int(w), 0)), borderCol, 1)
			g.Dummy(w, 1).Build()
		}),
		g.Dummy(0, 10),
		content,
		g.Dummy(0, 16),
	}
}

func rav3nToggle(id, label string, value *bool) g.Widget {
	return g.Custom(func() {
		const trackW, trackH = float32(44), float32(22)

		g.Style().SetColor(g.StyleColorText, colText).To(g.Label(label)).Build()
		g.SameLine()
		avail, _ := g.GetAvailableRegion()
		cursor := g.GetCursorPos()
		g.SetCursorPos(image.Pt(cursor.X+int(avail)-int(trackW), cursor.Y+1))

		localPos := g.GetCursorPos()
		screenPos := g.GetCursorScreenPos()

		on := *value
		canvas := g.GetCanvas()
		trackOff := color.RGBA{32, 32, 46, 255}
		trackColor := trackOff
		if on {
			trackColor = colAccent
		}
		canvas.AddRectFilled(
			screenPos,
			screenPos.Add(image.Pt(int(trackW), int(trackH))),
			trackColor, trackH/2, g.DrawFlagsRoundCornersAll,
		)
		if !on {
			canvas.AddRect(screenPos, screenPos.Add(image.Pt(int(trackW), int(trackH))), colBorder, trackH/2, g.DrawFlagsRoundCornersAll, 1)
		}
		knobX := float32(screenPos.X) + 11
		if on {
			knobX = float32(screenPos.X) + trackW - 11
		}
		canvas.AddCircleFilled(image.Pt(int(knobX), screenPos.Y+int(trackH/2)), 8, colText)

		g.SetCursorPos(localPos)
		g.InvisibleButton().ID("##toggle_" + id).Size(trackW, trackH).Build()
		if g.IsItemClicked(g.MouseButtonLeft) {
			*value = !*value
		}
		g.Dummy(-1, trackH+8).Build()
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
		g.Dummy(0, 2),
		g.SliderFloat(value, min, max).Size(-1),
		g.Dummy(0, 6),
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
	return g.Layout{
		g.Style().SetColor(g.StyleColorText, colText).To(g.Label(label)),
		g.Dummy(0, 4),
		g.Combo("##"+label, preview, items, selected).Size(-1).OnChange(onChange),
		g.Dummy(0, 6),
	}
}

func rav3nESPPreview() g.Widget {
	return g.Custom(func() {
		pos := g.GetCursorScreenPos()
		w, _ := g.GetAvailableRegion()
		h := float32(340)
		canvas := g.GetCanvas()

		canvas.AddRectFilled(pos, pos.Add(image.Pt(int(w), int(h))), color.RGBA{12, 12, 18, 255}, 12, g.DrawFlagsRoundCornersAll)
		canvas.AddRect(pos, pos.Add(image.Pt(int(w), int(h))), colBorder, 12, g.DrawFlagsRoundCornersAll, 1)

		cx := float32(pos.X) + w/2
		top := float32(pos.Y) + 44
		bottom := float32(pos.Y) + h - 36
		boxW := float32(76)
		left := int(cx - boxW/2)
		right := int(cx + boxW/2)
		topI := int(top)
		bottomI := int(bottom)

		boxCol := colTerrorist
		if TeamCheck {
			boxCol = colCT
		}
		fillCol := withAlpha(boxCol, 0.75)

		headY := top + 20
		neckY := top + 40
		chestY := top + 82
		pelvisY := top + 132
		previewBones := [][2]image.Point{
			{image.Pt(int(cx), int(headY)), image.Pt(int(cx), int(neckY))},
			{image.Pt(int(cx), int(neckY)), image.Pt(int(cx), int(chestY))},
			{image.Pt(int(cx), int(chestY)), image.Pt(int(cx), int(pelvisY))},
			{image.Pt(int(cx), int(pelvisY)), image.Pt(int(cx-18), int(bottom-10))},
			{image.Pt(int(cx), int(pelvisY)), image.Pt(int(cx+18), int(bottom-10))},
			{image.Pt(int(cx), int(neckY)), image.Pt(int(cx-28), int(chestY+20))},
			{image.Pt(int(cx-28), int(chestY+20)), image.Pt(int(cx-36), int(chestY+52))},
			{image.Pt(int(cx), int(neckY)), image.Pt(int(cx+28), int(chestY+20))},
			{image.Pt(int(cx+28), int(chestY+20)), image.Pt(int(cx+36), int(chestY+52))},
		}

		if BodyHighlightRendering {
			for _, seg := range previewBones {
				canvas.AddLine(seg[0], seg[1], fillCol, 14)
			}
			canvas.AddCircleFilled(image.Pt(int(cx), int(headY)), 14, fillCol)
		}
		if BoxRendering {
			drawPreviewCornerBox(canvas, left, topI, right, bottomI, boxCol, 2)
		}

		if HealthBarRendering {
			barX := left - 8
			canvas.AddRectFilled(
				image.Pt(barX, bottomI),
				image.Pt(barX+4, topI),
				color.RGBA{24, 24, 32, 255}, 2, g.DrawFlagsRoundCornersAll,
			)
			fillTop := topI + int(float32(bottomI-topI)*0.35)
			canvas.AddRectFilled(
				image.Pt(barX+1, bottomI-1),
				image.Pt(barX+3, fillTop),
				colSuccess, 2, g.DrawFlagsRoundCornersAll,
			)
		}

		if SkeletonRendering {
			skCol := withAlpha(boxCol, 0.85)
			headY := top + 20
			neckY := top + 40
			chestY := top + 82
			pelvisY := top + 132
			canvas.AddLine(image.Pt(int(cx), int(headY)), image.Pt(int(cx), int(pelvisY)), skCol, 2)
			canvas.AddLine(image.Pt(int(cx), int(neckY)), image.Pt(int(cx-28), int(chestY+20)), skCol, 2)
			canvas.AddLine(image.Pt(int(cx), int(neckY)), image.Pt(int(cx+28), int(chestY+20)), skCol, 2)
			canvas.AddLine(image.Pt(int(cx), int(pelvisY)), image.Pt(int(cx-18), int(bottom-10)), skCol, 2)
			canvas.AddLine(image.Pt(int(cx), int(pelvisY)), image.Pt(int(cx+18), int(bottom-10)), skCol, 2)
		}

		if HeadCircle {
			canvas.AddCircle(image.Pt(int(cx), int(top+20)), 15, withAlpha(boxCol, 0.5), 20, 1)
			canvas.AddCircle(image.Pt(int(cx), int(top+20)), 13, boxCol, 20, 2)
		}

		if NameRendering {
			tagW := 56
			canvas.AddRectFilled(
				image.Pt(int(cx)-tagW/2, topI-22),
				image.Pt(int(cx)+tagW/2, topI-6),
				color.RGBA{18, 18, 26, 230}, 4, g.DrawFlagsRoundCornersAll,
			)
			canvas.AddText(image.Pt(int(cx)-22, topI-20), colText, "Player")
		}

		g.Dummy(w, h).Build()
	})
}

func drawPreviewCornerBox(canvas *g.Canvas, left, top, right, bottom int, col color.RGBA, thickness float32) {
	cl := int(float32(right-left) * 0.22)
	if cl < 8 {
		cl = 8
	}
	if cl > 16 {
		cl = 16
	}
	draw := func(x1, y1, x2, y2 int) {
		canvas.AddLine(image.Pt(x1, y1), image.Pt(x2, y2), col, thickness)
	}
	draw(left, top, left+cl, top)
	draw(left, top, left, top+cl)
	draw(right, top, right-cl, top)
	draw(right, top, right, top+cl)
	draw(left, bottom, left+cl, bottom)
	draw(left, bottom, left, bottom-cl)
	draw(right, bottom, right-cl, bottom)
	draw(right, bottom, right, bottom-cl)
}

func rav3nStatusBar() g.Widget {
	return g.Custom(func() {
		canvas := g.GetCanvas()
		pos := g.GetCursorScreenPos()
		w, _ := g.GetAvailableRegion()
		h := float32(28)
		canvas.AddRectFilled(pos, pos.Add(image.Pt(int(w), int(h))), colSidebar, 0, 0)
		canvas.AddLine(pos, pos.Add(image.Pt(int(w), 0)), colBorder, 1)

		dotCol := colSuccess
		canvas.AddCircleFilled(pos.Add(image.Pt(14, 14)), 4, dotCol)

		g.SetCursorPos(image.Pt(26, 6))
		g.Style().SetColor(g.StyleColorText, colTextMuted).To(
			g.Label("Ready  ·  CS2 External  ·  Offsets via cs2-dumper"),
		).Build()

		g.SetCursorPos(image.Pt(int(w)-120, 6))
		g.Style().SetColor(g.StyleColorText, colTextDim).To(g.Label("RAV3N v2.1")).Build()
		g.Dummy(-1, h).Build()
	})
}

func rav3nPageTitle(title, subtitle string) g.Widget {
	return g.Layout{
		g.Style().SetColor(g.StyleColorText, colText).To(g.Label(title)),
		g.Dummy(0, 4),
		g.Custom(func() {
			pos := g.GetCursorScreenPos()
			w, _ := g.GetAvailableRegion()
			canvas := g.GetCanvas()
			canvas.AddRectFilled(pos, pos.Add(image.Pt(48, 3)), colAccent, 2, g.DrawFlagsRoundCornersAll)
			g.Dummy(w, 3).Build()
		}),
		g.Dummy(0, 6),
		g.Style().SetColor(g.StyleColorText, colTextMuted).To(g.Label(subtitle)),
		g.Dummy(0, 16),
	}
}

func rav3nPerfGraph() g.Widget {
	return g.Custom(func() {
		pos := g.GetCursorScreenPos()
		w, _ := g.GetAvailableRegion()
		h := float32(140)
		canvas := g.GetCanvas()

		canvas.AddRectFilled(pos, pos.Add(image.Pt(int(w), int(h))), color.RGBA{14, 14, 20, 255}, 8, g.DrawFlagsRoundCornersAll)
		canvas.AddRect(pos, pos.Add(image.Pt(int(w), int(h))), colBorder, 8, g.DrawFlagsRoundCornersAll, 1)

		pad := float32(12)
		graphLeft := float32(pos.X) + pad
		graphRight := float32(pos.X) + w - pad
		graphTop := float32(pos.Y) + pad + 8
		graphBottom := float32(pos.Y) + h - pad
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

		canvas.AddText(image.Pt(int(pos.X)+12, int(pos.Y)+8), colTextMuted, "Performance Monitor")
		canvas.AddText(image.Pt(int(pos.X)+12, int(pos.Y)+int(h)-22), colAccent, fmt.Sprintf("GUI %3.0f FPS", perfGuiFPS))
		canvas.AddText(image.Pt(int(pos.X)+120, int(pos.Y)+int(h)-22), colSuccess, fmt.Sprintf("Overlay %5.1f ms", perfOverlayMs))

		g.Dummy(w, h).Build()
	})
}
