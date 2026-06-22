package main

const perfHistorySize = 120

var (
	perfGuiFPS         float32
	perfOverlayMs      float32
	perfGuiHistory     = make([]float32, 0, perfHistorySize)
	perfOverlayHistory = make([]float32, 0, perfHistorySize)
)

func recordGuiFrame(fps float32) {
	if fps <= 0 {
		return
	}
	perfGuiFPS = fps
	perfGuiHistory = append(perfGuiHistory, fps)
	if len(perfGuiHistory) > perfHistorySize {
		perfGuiHistory = perfGuiHistory[1:]
	}
}

func recordOverlayFrame(ms float32) {
	if ms < 0 {
		return
	}
	perfOverlayMs = ms
	perfOverlayHistory = append(perfOverlayHistory, ms)
	if len(perfOverlayHistory) > perfHistorySize {
		perfOverlayHistory = perfOverlayHistory[1:]
	}
}
