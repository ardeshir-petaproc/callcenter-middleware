# Call Center Desktop

A native desktop client (Electron) for agents and supervisors. It wraps the
web control panel served by the call center middleware
(`http://<server>:8080/`) and adds:

- a first-run **connection screen** to pick the server (validated with `GET /healthz`)
- a **tray icon** with a status quick-switch (Available / Paused / Offline), launch-at-login, change server, quit
- **native notifications** for incoming calls (`agent.screenpop`), plus taskbar flashing / dock bounce
- **close to tray**, so the softphone and the realtime connection keep running in the background
- microphone access for the panel's WebRTC softphone (and nothing else)
- a single instance, remembered window size and position, zoom, reload

## Quick start (development)

```bash
cd desktop
npm install            # see "Offline / proxied installs" if the Electron download fails
npm start              # tsc build, then electron .
CC_DEBUG=1 npm start   # also enables DevTools (View → Toggle Developer Tools) and renderer console logging
```

On first launch a small window asks for the server address, for example
`http://pbx.example.com:8080` or `10.0.0.5:8080` (`http://` is assumed when no
scheme is given). **Test connection** calls `<url>/healthz`. **Connect** saves
the address and opens the panel. If the health check fails you can still save
with **Connect anyway**, for example when the server is down for maintenance.

To change the server later, use **File → Change Server…** or **Change server…** in the tray menu.

## Configuration

Settings are stored as JSON in the Electron `userData` directory:

| OS      | Path |
|---------|------|
| Linux   | `~/.config/Call Center/settings.json` (or `$XDG_CONFIG_HOME/...`) |
| Windows | `%APPDATA%\Call Center\settings.json` |
| macOS   | `~/Library/Application Support/Call Center/settings.json` |

```json
{
  "serverUrl": "http://pbx.example.com:8080",
  "closeToTray": true,
  "notifyIncomingCalls": true,
  "window": { "x": 100, "y": 80, "width": 1280, "height": 820, "maximized": false }
}
```

The tray menu changes `closeToTray` and `notifyIncomingCalls`. Unknown or invalid fields are ignored.

Environment variables and flags:

| Name | Effect |
|------|--------|
| `CC_SERVER_URL=<url>` or `--server=<url>` | Sets the server on **first run** only, so the setup screen is skipped. This is useful for mass deployment. Once a server is saved, these are ignored. |
| `CC_DEBUG=1` | Enables DevTools, the DevTools menu item and forwarding of renderer console messages to stdout. |
| `--hidden` | Starts straight to the tray. The launch-at-login entry uses this. |

## Features in detail

### Tray

- **Show Call Center** opens the window (a left click on the tray icon also does this on Windows and macOS).
- The tray shows who is signed in and their live status (for example "Alice — On a call"). The icon has a status dot: green = available, amber = paused, grey = offline, blue = ringing, red = on a call, purple = wrap-up.
- **Available / Paused / Offline** and **Pause with reason ›** (listed from `GET /api/v1/pause-reasons`) change your own status.
- **Notify on incoming calls**, **Close window to tray** and **Launch at login** are on/off toggles.
- **Change server…**, **Reload panel** and **Quit** are also in the tray menu.

#### How the status quick-switch works

The main process calls **`POST /api/v1/me/status`** (`{status, reason?}`) directly. It authenticates with the bearer token that the panel keeps in `localStorage["cc_token"]`, which it reads with `webContents.executeJavaScript`.

We chose this over injecting a call into the page because it does not depend on the panel's internal JavaScript. It works the same whatever React state, route or version the panel has, and even while the window is hidden. It is the documented API that the panel itself uses. The panel receives the resulting `agent.updated` event over its own WebSocket, so its UI stays in sync. If the call fails, for example because the token expired, an error dialog explains why.

### Incoming call notifications

The main process keeps its **own** WebSocket to `<server>/api/v1/ws?token=<cc_token>`:

1. Every 3 s, and after each page load, the app reads `cc_token` from the panel's localStorage. A new token triggers `GET /api/v1/auth/me` to learn the user id. A missing token means logged out, so the socket closes. A token rejected with 401 is not retried until it changes.
2. On `agent.screenpop` where `data.user_id` is the signed-in user, the app shows a native notification. It has the caller name and number and the queue name, which comes from the queue list in the periodic `snapshot` message. Clicking it brings the window to the front. If the window is not focused, the taskbar entry flashes (Windows/Linux) or the dock icon bounces (macOS).
3. The app does not notify when the panel window is focused, because the panel shows its own screen pop. Repeated screenpops for the same call within 60 s produce a single notification.
4. The socket reconnects with exponential backoff and jitter (1 s → 30 s). If it receives nothing for 45 s it reconnects; the server sends a snapshot every 10 s. After a disconnect it re-checks `/auth/me`, so an expired token stops the reconnect loop.

The client uses the global `WebSocket` of Electron's Node runtime (Electron 45 ships Node 24). The `ws` package is kept as a fallback for runtimes without one.

### Bridge for the web panel: `window.ccDesktop`

When the panel runs inside the desktop app it sees a small frozen object:

```ts
interface CcDesktop {
  version: string;                 // desktop app version, e.g. "0.1.0"
  platform: string;                // "linux" | "win32" | "darwin"
  notify(title: string, body?: string): void;  // native notification; click focuses the app
  setBadge(count: number): void;   // dock/launcher badge (macOS, Linux Unity) or taskbar overlay (Windows); 0 clears
  focus(): void;                   // show + restore + focus the window
}
```

The panel can feature-detect it with `if (window.ccDesktop) { ... }`. For example, it can call `setBadge` with the number of unheard voicemails or pending callbacks. `notify` is rate-limited to 5 notifications per 10 s.

### Window

- The app remembers the window's size, position and maximized state. A position that is no longer on any screen is discarded.
- **Close** hides the window to the tray, unless "Close window to tray" is off or no tray is available. **Quit** is in the tray and File menus (Ctrl/Cmd+Q).
- The **View** menu has Reload (Ctrl/Cmd+R), Force Reload, zoom in/out/reset (Ctrl/Cmd +/−/0) and full screen. Developer Tools are available only with `CC_DEBUG=1`.
- The menu bar auto-hides on Windows and Linux; press Alt to show it.
- If the panel cannot be loaded, a local "server unreachable" page appears and retries every 15 s.

## Security model

The panel is remote content, so it is treated as untrusted with respect to the local machine:

- **Renderer isolation.** Every window has `contextIsolation: true`, `nodeIntegration: false`, `sandbox: true` and `webviewTag: false`. `app.enableSandbox()` is also called. The page cannot reach Node, `require` or `ipcRenderer`.
- **Minimal bridge.** The preload exposes only the four `ccDesktop` functions above, through `contextBridge`, as a frozen object. The bridge is exposed only when the main process confirms that the document is on the configured server origin. Each IPC message is checked again in the main process: it must come from the main window's top-level frame, and that frame must be on the server origin. Arguments are coerced to strings or numbers and truncated.
- **Navigation lock.** The main window may only navigate within the configured server origin. Other links, as well as all `window.open` and `target=_blank` links, open in the default browser, and only for `http:`, `https:` and `mailto:`. Other schemes are dropped. `<webview>` attachment is blocked everywhere.
- **Permissions.** The panel runs in its own persistent session partition (`persist:panel`), which has these handlers:
  - `media` is granted only to the configured server origin and only for **audio** (microphone). Camera requests are denied. On macOS the app also asks for the OS microphone permission.
  - `clipboard-sanitized-write` is granted only to the server origin, so the panel's "copy" buttons work. This is write-only.
  - Every other permission request is denied and logged, including notifications, geolocation, camera, screen capture, MIDI, HID, serial and USB.
- **Secure context for plain-HTTP servers.** `getUserMedia` requires a secure context. When the configured server is `http://` on a non-loopback host, the app starts Chromium with `--unsafely-treat-insecure-origin-as-secure=<that exact origin>`. This applies to one origin only, and is what lets the softphone use the microphone on an internal HTTP server. Command-line switches only take effect at startup, so changing to such a server relaunches the app once. HTTPS is still strongly recommended.
- **Local pages.** The setup and offline pages are local files with a strict CSP (`default-src 'none'; script-src 'self'; ...`) and no inline scripts. They have their own preload with just `info / check / save / cancel`.
- **Token handling.** The token is read from the panel's own storage and used only for `/auth/me`, `/me/status`, `/pause-reasons` and the WebSocket. It is never written to disk by the desktop app and never logged; it is shown as `token=***` in logs.
- **TLS.** REST calls use Electron's `net.fetch`, which goes through the Chromium network stack and so honours the OS certificate store and proxy settings. The background WebSocket uses Node's TLS, so for a private CA also set `NODE_EXTRA_CA_CERTS=/path/to/ca.pem`. Certificate errors are never bypassed.
- **Outbound connections.** Spellcheck is on. On Linux, Chromium downloads Hunspell dictionaries from Google's CDN the first time. If that is not allowed, it fails harmlessly.

## Packaging

[electron-builder](https://www.electron.build/) settings are in `package.json` under `"build"`.

```bash
npm run dist:linux   # AppImage + .deb  → release/
npm run dist:win     # NSIS installer   (build on Windows, or Linux with wine)
npm run dist:mac     # DMG              (build on macOS; signing/notarization not configured)
npm run dist         # targets for the current OS
```

Each `dist*` script compiles TypeScript (`npm run build`), then generates `build/icon.png` (`npm run icons`, see below), then runs electron-builder.

Icons are not committed as binaries. `src/icon.ts` contains a tiny anti-aliased rasterizer and a PNG encoder based on `node:zlib`. The app uses it at runtime for the tray icon with its status dot, the window icon and the Windows overlay badge. `scripts/gen-icons.js` uses it at build time to write the 512×512 `build/icon.png` that electron-builder converts to `.ico` and `.icns`.

Before shipping, also:

- Replace the placeholder `author`, `homepage` and `build.linux.maintainer` email (`example.invalid`) in `package.json`.
- Set up code signing (Windows `CSC_LINK`, macOS Developer ID and notarization). The macOS entitlements in `build/entitlements.mac.plist` already include `audio-input` and the `NSMicrophoneUsageDescription` usage string.
- Note that the deb installs the binary as `callcenter-desktop`.

### Offline / proxied installs

Electron 45 downloads its binary from GitHub when you install it. If that is blocked, use one of these options:

```bash
ELECTRON_SKIP_BINARY_DOWNLOAD=1 npm install   # enough for typecheck/build
# or fetch electron-v45.0.0-<platform>-<arch>.zip from a mirror:
ELECTRON_MIRROR=https://your.mirror/electron/ npx install-electron
```

electron-builder can reuse an already extracted Electron via `-c.electronDist=node_modules/electron/dist`.

## Project layout

```
desktop/
├── package.json             scripts, pinned deps, electron-builder config
├── tsconfig.json            main + preload (src → dist, CommonJS)
├── tsconfig.renderer.json   type-checks static/*.js (checkJs, DOM lib)
├── src/
│   ├── main.ts              app lifecycle, windows, tray, IPC, permissions, notifications
│   ├── session.ts           token → /auth/me → WebSocket; status changes; screenpop filtering
│   ├── realtime.ts          reconnecting WebSocket (global WebSocket or `ws`)
│   ├── server.ts            URL normalization, /healthz, REST helper, API types
│   ├── menus.ts             application + tray menus
│   ├── settings.ts          JSON settings store (atomic writes)
│   ├── autostart.ts         launch at login (login items / XDG autostart)
│   ├── icon.ts              programmatic icon + PNG encoder
│   └── preload/
│       ├── panel.ts         window.ccDesktop bridge (remote panel)
│       └── setup.ts         window.ccSetup bridge (local setup page)
├── static/                  setup.html/js, offline.html/js, ui.css (light/dark)
├── scripts/                 gen-icons.js, check-syntax.js
└── build/                   entitlements.mac.plist (+ generated icon.png)
```

## Scripts

| Script | What it does |
|--------|--------------|
| `npm start` | Build, then run the app. |
| `npm run build` | `tsc` → `dist/`. |
| `npm run typecheck` | Type-check `src/` (TS) and `static/*.js` (`// @ts-check`). |
| `npm run check` | Typecheck, build, then `node --check` every JS file. |
| `npm run icons` | Generate `build/icon.png`. |
| `npm run dist`, `dist:linux`, `dist:win`, `dist:mac` | Package installers into `release/`. |

## Limitations

- **Launch at login** uses `app.setLoginItemSettings` on Windows and macOS. On Linux it writes `~/.config/autostart/callcenter-desktop.desktop`; for AppImages it points at `$APPIMAGE`.
- **Linux tray.** On GNOME the tray icon needs the AppIndicator extension. Without it the icon is invisible. In that case, turn off **Close window to tray** (File menu) and don't use launch-at-login, which starts the app hidden. If Electron cannot create a tray at all, closing the window quits the app and `--hidden` is ignored.
- **Notification styling** depends on the OS. On Linux a notification daemon is required, and urgency is set to *critical*.
- **Status display.** The tray reflects statuses reported by the server, including ringing, busy and wrap-up. Only Available, Paused and Offline can be set from the tray.
