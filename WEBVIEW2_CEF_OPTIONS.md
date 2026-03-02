# WebView2 Instantiation + CEF Mapping

Checked against: https://github.com/wailsapp/go-webview2 (module `github.com/wailsapp/go-webview2@v1.0.23`, commit `e08e9397f04ad0b1050d7e5bdc8e27faddc87987`, fetched on 2026-03-02).

## 1) Structs involved in creating a WebView2 instance

| Struct/type | Location | Purpose | Required to create instance? |
|---|---|---|---|
| `edge.Chromium` | `pkg/edge/chromium.go` | Main user-facing WebView2 wrapper object. | Yes |
| `webviewloader.environmentOptions` | `webviewloader/env_create_options.go` | Internal environment config container used by loader options. | Yes (internally, defaults are valid) |
| `webviewloader.option` | `webviewloader/env_create_options.go` | Functional options type applied to `environmentOptions`. | Optional |

## 2) Required vs optional values for instantiation

### Required (practical)

- `hwnd uintptr` passed to `(*edge.Chromium).Embed(hwnd)` must be a valid native window handle.
- A resolvable WebView2 runtime must exist: either installed Evergreen runtime, or a valid fixed runtime path via `BrowserPath` / `WithBrowserExecutableFolder`.

### Optional (user-configurable)

#### On `edge.Chromium` (before `Embed`)

- `Debug bool` (present on struct; not used by `Embed` path directly)
- `DataPath string` (user data folder)
- `BrowserPath string` (fixed runtime folder)
- `AdditionalBrowserArgs []string` (joined and passed to environment creation)

#### `webviewloader` functional options (`CreateCoreWebView2EnvironmentWithOptions`)

- `WithBrowserExecutableFolder(folder string)`
- `WithUserDataFolder(folder string)`
- `WithAdditionalBrowserArguments(args string)`
- `WithLanguage(lang string)`
- `WithTargetCompatibleBrowserVersion(version string)`
- `WithAllowSingleSignOnUsingOSPrimaryAccount(allow bool)`
- `WithExclusiveUserDataFolderAccess(exclusive bool)`

## 3) CEF options vs WebView2 options map

| WebView2 option | CEF equivalent | Adapter status in this repo |
|---|---|---|
| `BrowserPath` / `WithBrowserExecutableFolder` | No direct CEF runtime-folder equivalent in current CEF wrapper | Exposed in adapter config as compatibility field; currently warned as unmapped |
| `DataPath` / `WithUserDataFolder` | Closest concept is CEF cache/user-data path | Exposed in adapter config as compatibility field; currently warned as unmapped |
| `AdditionalBrowserArgs` / `WithAdditionalBrowserArguments` | Chromium command-line args | Mapped: adapter appends args to `os.Args` before `cef.Initialize()` |
| `WithLanguage` | `--lang=...` command-line switch | Mapped: adapter appends `--lang=<value>` |
| `WithTargetCompatibleBrowserVersion` | No direct CEF equivalent | Exposed as compatibility field; currently warned as unmapped |
| `WithAllowSingleSignOnUsingOSPrimaryAccount` | No direct CEF equivalent at current wrapper level | Exposed as compatibility field; currently warned as unmapped |
| `WithExclusiveUserDataFolderAccess` | No direct CEF equivalent at current wrapper level | Exposed as compatibility field; currently warned as unmapped |
| `Embed(hwnd)` | Native parent window binding in CEF window creation | Not directly user-configurable in adapter API; handled internally by CEF backend |

## 4) Current adapter fields added for WebView2 parity

In `adapter/webview/webview.go`, `AppConfig` now includes:

- Core window options: `Resizable`, `Fullscreen`, `Maximized`, `X`, `Y`
- WebView2-compat fields: `DataPath`, `BrowserPath`, `AdditionalBrowserArgs`, `Language`, `TargetCompatibleBrowserVersion`, `AllowSingleSignOnUsingOSPrimaryAccount`, `ExclusiveUserDataFolderAccess`

Mapped behavior is applied in `webview.applyWebView2CompatOptions()` before CEF init.

## 5) Source pointers used

- `pkg/edge/chromium.go`
- `webviewloader/env_create.go`
- `webviewloader/env_create_options.go`
- Repository URL: https://github.com/wailsapp/go-webview2
