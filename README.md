# Dejima KVM

Named after [Dejima](https://en.wikipedia.org/wiki/Dejima), the small island in Nagasaki that served as Japan's sole gateway to the outside world during its period of national isolation (1641–1853). Just as Dejima bridged two isolated worlds through a narrow physical channel, **Dejima KVM** lets you operate a network-isolated machine through a purely physical connection — HDMI for video and serial UART for input — with no network link required.

A software KVM switch that combines USB HDMI capture video display with serial HID control, allowing you to remotely operate a Raspberry Pi 4 from your host PC.

## Architecture

```
Host PC (Mac/Windows)              RPi4
┌──────────────────┐          ┌──────────────┐
│  dejima-kvm      │          │  dejima-kvm  │
│  (Wails app)     │          │  -daemon-rpi │
│                  │  HDMI    │              │
│  Video ◄─────────┼──────────┤  HDMI out    │
│  (USB capture)   │          │              │
│                  │  UART    │              │
│  Keyboard/Mouse ─┼──────────►  uinput     │
│  (FT232 serial)  │          │              │
└──────────────────┘          └──────────────┘
```

- **Video path**: RPi4 HDMI output → USB capture device → GStreamer → MJPEG stream in app
- **HID path**: Keyboard/mouse input → serial framing → FT232 UART → RPi4 → uinput virtual devices

## Hardware Requirements

| Component | Purpose |
|-----------|---------|
| USB HDMI capture device | Display RPi4 screen on host PC |
| FT232 USB-Serial adapter (3.3V) | Send keyboard/mouse input over serial |
| Raspberry Pi 4 | Target machine to control |
| HDMI cable | RPi4 → capture device |

## Software Components

| Component | Description |
|-----------|-------------|
| `dejima-kvm` | Host-side GUI application (Wails v2 + Svelte + TypeScript) |
| `dejima-kvm-daemon-rpi` | RPi4-side daemon (Go + uinput) |

## Quick Start

### Host PC (macOS)

```bash
# Install dependencies
brew install gstreamer gst-plugins-base gst-plugins-good

# Development mode
wails dev

# Production build
wails build
```

### RPi4

```bash
# Cross-compile on host PC
make build-rpi

# Deploy to RPi4
scp build/bin/dejima-kvm-daemon-rpi pi@<rpi-ip>:/usr/local/bin/
scp rpi-daemon/dejima-kvm-rpi.service pi@<rpi-ip>:/etc/systemd/system/
ssh pi@<rpi-ip> 'sudo systemctl enable --now dejima-kvm-rpi'
```

## Usage

1. Connect USB HDMI capture device and FT232 adapter to host PC
2. Launch the app and select **Video > Start** to begin video capture
3. Open **Serial** menu, select port, and connect
4. Click the video area to start input capture — keyboard and mouse are forwarded to RPi4
5. Press **Esc** to release input capture

## Documentation

| File | Contents |
|------|----------|
| [docs/usage-mac.md](docs/usage-mac.md) | Detailed usage guide for macOS |
| [docs/usage-windows.md](docs/usage-windows.md) | Usage guide for Windows |
| [docs/setup-rpi4.md](docs/setup-rpi4.md) | RPi4 setup guide |
| [docs/shortcuts-mac.md](docs/shortcuts-mac.md) | macOS keyboard shortcut reference |

## License

[MIT](LICENSE)
