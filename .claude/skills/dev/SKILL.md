---
name: dev
description: Kill leftover processes and start Wails dev server (Windows)
---

# Start Wails Dev Server (Windows)

Cleanly restart the development environment by killing stale processes first.

## Steps

1. Kill any existing dejima / GStreamer processes (ignore errors if none found):

```bash
wmic process where "ExecutablePath like '%dejima%'" get ProcessId 2>/dev/null | grep -o '[0-9]*' | xargs -r -I{} taskkill //F //PID {} //T 2>/dev/null; taskkill //F //IM "gst-launch-1.0.exe" 2>/dev/null; rm -f "$APPDATA/dejima/app.lock"; sleep 2
```

2. Start `wails dev` in the background from the project root:

```bash
cd /c/home/itosaygo/doc/1_products/2026/dev/dejima && wails dev
```

Run this command with `run_in_background: true` so the user can keep working.

3. Wait up to 90 seconds for the build to complete, then confirm the app launched by checking for "WebView2" or "stream server" in the output.
