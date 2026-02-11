# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Dejima KVM is a software KVM switch for remotely operating a Raspberry Pi 4 from a host PC (macOS/Windows) through purely physical connections — HDMI for video capture and serial UART (FT232) for keyboard/mouse input. No network link is required.

## Build & Development Commands

```bash
# Development (hot reload)
make dev                    # or: wails dev

# Production build
make build                  # current OS
make build-windows          # cross-compile for Windows

# RPi daemon (cross-compile to ARM64 Linux)
make build-rpi

# Frontend dependencies
make install-deps           # cd frontend && npm install

# Frontend type checking
cd frontend && npm run check   # svelte-check + TypeScript validation
```

There are no automated tests in this project.

## Architecture

Two separate Go binaries communicate over FT232 serial UART:

### Host App (`main.go`, `app.go`)
Wails v2 desktop app (Go backend + Svelte/TypeScript frontend). The `App` struct in `app.go` is the central API surface — its public methods are bound to the frontend via Wails.

### RPi Daemon (`rpi-daemon/`)
Standalone Go binary that reads serial frames and injects HID events via Linux uinput. Has its own `go.mod` and `internal/` packages.

### Data Flow
```
Frontend DOM events → keyboard.ts/mouse.ts → App methods → HID Controller
  → Protocol encode → Serial write → FT232 UART → RPi daemon → uinput

RPi HDMI out → USB capture → GStreamer subprocess → FrameStore → MJPEG HTTP → Frontend <img>
```

## Key Internal Packages (`internal/`)

- **protocol/** — Binary message encoding/decoding (11 message types). Shared concept between host and RPi daemon (each has its own copy).
- **serial/** — FT232 serial port management with STX/LEN/CHECKSUM/ETX framing (`transport.go`), platform-specific auto-detection (`detector.go`).
- **video/** — GStreamer subprocess management (`gstreamer.go`), thread-safe JPEG frame buffer (`frame.go`), MJPEG HTTP handler. Platform-specific pipeline construction in `pipeline_darwin.go`, `pipeline_linux.go`, `pipeline_windows.go`.
- **audio/** — GStreamer capture + oto playback, also platform-specific.
- **hid/** — Routes frontend input events through protocol encoder. Contains JS→Linux key code mapping (`keyboard.go`).
- **config/** — JSON config file stored at `~/.config/dejima/config.json` (or `%APPDATA%\dejima\` on Windows).

## Frontend (`frontend/src/`)

Svelte 3 + TypeScript + Vite. Key areas:

- **`lib/input/keyboard.ts`** — Keyboard capture with Cmd→Ctrl translation for macOS→Linux, clipboard sync hooks (Cmd+V/C).
- **`lib/input/mouse.ts`** — Mouse capture with relative/absolute positioning modes.
- **`lib/stores/`** — Svelte stores for connection state and IME mode.
- **`lib/components/`** — UI components (VideoDisplay, StatusBar, settings modals).

## Platform Considerations

Video/audio device selection and GStreamer pipelines differ per platform. Each has a dedicated `pipeline_<os>.go` file. Serial FT232 detection is also platform-specific (`detector.go` with build tags). When modifying video, audio, or serial code, check all platform variants.

## Tech Stack

- **Backend**: Go 1.24, Wails v2, go.bug.st/serial, ebitengine/oto
- **Frontend**: Svelte 3, TypeScript 4.6, Vite 3
- **RPi Daemon**: Go 1.24, bendahl/uinput
- **System**: GStreamer (video/audio capture), FT232 UART (HID transport)
