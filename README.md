# EXPERIMENTAL NOT RELEASED YET

# RAV3N

RAV3N is a Windows external cheat for Counter-Strike 2 written in Go. It reads game memory externally and renders ESP visuals. This project is based on [cs2go](https://github.com/NYPDK/cs2go).

> **Disclaimer:** This project is provided for educational and research purposes only. Using third-party tools in online games may violate the game's Terms of Service and result in a ban. Use at your own risk.

## Features

### Combat
- **Aimbot** — hold-to-activate targeting with configurable FOV, smoothing, activation key, and bone selection (head, neck, chest, pelvis)
- **Recoil control** — per-weapon presets (Default, AK-47, M4A4, M4A1-S, Galil AR, FAMAS) with customizable compensation axes and smoothing

### Visuals (ESP)
- Bounding boxes (team-colored corner boxes)
- Body highlight (filled bone silhouette)
- Skeleton overlay
- Head circle
- Player names
- Health bar and health text
- Team filter (hide teammates)
- Live ESP preview panel in the GUI

### Performance
- Target overlay refresh rate capped at 120 FPS
- Configurable minimum frame time to reduce CPU usage
- Batch memory reads, entity-list stride caching, and reused GDI back-buffer
- Real-time performance monitor (GUI FPS + overlay frametime graph)

### Configuration
- Save / load / delete JSON profiles in `configs/`
- Export / import named profiles to `configs/exports/`
- Three UI theme presets: Raven Purple, Crimson Elite, Ice Neon

### Offsets
- Automatic download from [a2x/cs2-dumper](https://github.com/a2x/cs2-dumper) on every launch (online)
- Local cache in `offsets.json` (used when offline, refreshed if younger than 1 hour)
- Offline fallback to the last cached offsets file

## Requirements

| Requirement | Details |
|---|---|
| OS | Windows 10 / 11 (64-bit) |
| Game | Counter-Strike 2 (`cs2.exe`) running |
| Go | 1.22 or newer |
| GCC | [MinGW-w64](https://www.mingw-w64.org/) — required for the GUI (CGO) |
| Network | Internet access on first launch (offset download) |

## Build

All source files share `package main` and are compiled into a single binary:

```bash
go mod tidy
go build -o rav3n.exe .
```

If the build fails with `C compiler "gcc" not found`, install MinGW-w64 and ensure `gcc` is available in your `PATH`.

## Usage

1. Launch Counter-Strike 2.
2. Run `rav3n.exe` from the project directory (or copy the executable next to its runtime files).
3. On first launch, offsets are fetched automatically and saved to `offsets.json`.
4. Use the RAV3N GUI to configure features. The overlay renders on top of the game window.
5. Close the application from the GUI **Exit Application** button or the title bar **×** button.

### GUI sections

| Section | Description |
|---|---|
| **Aimbot** | Enable aimbot, set FOV, smoothing, activation key, target bone, and recoil control |
| **Player ESP** | Toggle ESP elements and preview the overlay layout |
| **Settings** | Frame pacing, theme preset, performance monitor, about info |
| **Configs** | Profile management, named export/import, theme selection |

### Default aimbot key

Mouse 5 (`VK_XBUTTON2`, `0x06`) — changeable in the GUI under **Activation Key**.

## Architecture

RAV3N is a single Go module (`rav3n`) split into focused source files. There are no sub-packages: every `.go` file is part of `package main` and links at compile time via `go build`.

```
main()
 ├── loadOffsets()          offsets.go
 ├── go runOverlay()        main.go  ─┐
 └── RunGui()               gui.go    │
                                       ▼
                              ┌────────────────────┐
                              │   Overlay goroutine │
                              └─────────┬──────────┘
                                        │
          findProcessId / getModuleBaseAddress / readPtr
                                        │  memory.go
          findGameWindow / syncOverlayToGame
                                        │  game_window.go
          readViewProjection / getEntitiesInfo
                                        │  entities.go
          aimbot()                      │  aimbot.go
          recoilControl()                 │  main.go
          renderEntity()                  │  esp_render.go
          recordOverlayFrame()            │  gui_perf.go
```

### Shared globals

Combat and ESP toggles (`AimbotEnabled`, `BoxRendering`, `TeamCheck`, etc.) are declared in `main.go` and read by the overlay loop, aimbot, entity iteration, and GUI pages. Profile save/load in `gui.go` snapshots and restores these values.

## Project structure

```
rav3n/
├── main.go           Entry point, Win32 overlay window, overlay loop, recoil control
├── aimbot.go         Aimbot targeting, smoothing, FOV lock, bone averaging
├── entities.go       Entity types, world-to-screen, bone reads, player iteration
├── esp_render.go     GDI drawing: boxes, skeleton, body highlight, health, names
├── game_window.go    CS2 window discovery, client-area tracking, overlay sync
├── memory.go         Process/module access, NtReadVirtualMemory, entity list helpers
├── offsets.go        cs2-dumper fetch, schema parsing, offsets.json cache
├── gui.go            GUI layout, pages, profile save/load/export
├── gui_theme.go      Color palette, theme presets, UI animation state
├── gui_widgets.go    Custom UI components (cards, toggles, nav, ESP preview, graphs)
├── gui_perf.go       FPS / frametime history for the performance monitor
├── configs/          Saved profiles (created at runtime)
├── configs/exports/  Named export profiles (created at runtime)
└── offsets.json      Cached game offsets (created at runtime)
```

| File | Role |
|---|---|
| `main.go` | `main()`, overlay lifecycle, GDI setup, frame loop orchestration |
| `aimbot.go` | Target selection, hysteresis deadzone, mouse movement |
| `entities.go` | `Entity` struct, view matrix projection, player enumeration |
| `esp_render.go` | All on-screen ESP rendering via Win32 GDI |
| `game_window.go` | EnumWindows scan to find and track the CS2 client area |
| `memory.go` | External memory reads, entity list stride (`0x70`) |
| `offsets.go` | Live offset download + `offsets.json` persistence |
| `gui.go` | Dear ImGui window via giu, config pages, JSON profiles |
| `gui_theme.go` | Theme colors, easing helpers, `uiAnim` state |
| `gui_widgets.go` | Reusable widgets: header, sidebar, cards, sliders, preview |
| `gui_perf.go` | Rolling history buffers for the perf graph |

## Profiles

Profiles store all combat, visual, and theme settings as JSON.

| Action | Location |
|---|---|
| Save / Load / Delete | `configs/<name>.json` |
| Export / Import | `configs/exports/<name>.json` |

Profile names are sanitized to alphanumeric characters, underscores, and hyphens.

To force a fresh offset download after a CS2 game update, delete `offsets.json` and relaunch RAV3N with an internet connection.

## Performance tuning

- Increase **Min Frame Time (ms)** in Settings to lower CPU usage (higher value = fewer overlay updates per second).
- Disable skeleton or body highlight ESP if you only need boxes — both read full bone data per player.
- The performance monitor in Settings shows GUI FPS (purple) and overlay throughput (green, derived from frametime).

## Troubleshooting

| Issue | Solution |
|---|---|
| ESP not showing | Ensure CS2 is running before or after launch; delete `offsets.json` and relaunch |
| Build fails (gcc) | Install MinGW-w64 and add it to `PATH` |
| Stale offsets | Delete `offsets.json` or wait for the 1-hour cache to expire |
| No profiles listed | Save a profile first — the `configs/` folder is created automatically |
| Overlay misaligned | RAV3N tracks the CS2 client area each frame via `game_window.go` |

## Dependencies

- [giu](https://github.com/AllenDang/giu) — GUI framework (Dear ImGui)
- [lxn/win](https://github.com/lxn/win) — Win32 overlay rendering
- [golang.org/x/sys/windows](https://pkg.go.dev/golang.org/x/sys/windows) — Windows syscalls
- [a2x/cs2-dumper](https://github.com/a2x/cs2-dumper) — live game offsets

## License

See [LICENSE](LICENSE).
