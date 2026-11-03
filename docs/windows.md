# Windows

Patch targets **Windows x64 (Intel/AMD)**. The terminal requires **Windows 10
version 1809 or later**, including Windows 11, for ConPTY. Install **Git 2.36 or
later** and put it on `PATH` to use the Git panel.

The optional Browser panel needs Microsoft's
[WebView2 Runtime](https://developer.microsoft.com/microsoft-edge/webview2/),
included with Windows 11. The native interface and terminal do not need it
([MyGo requirements](https://github.com/egoist/mygo/blob/6842ad5e0aa2/docs/README.md#requirements)).

## Download and run

Each push to `main` runs **Windows portable build** in GitHub Actions. Open the
successful run and download its `patch-0.1.0-windows-amd64.zip` artifact (the version
follows `mygo.json`). Extract the entire ZIP into a folder, then open `Patch.exe`.
Keep `ghostty-vt.dll` and the `fonts` directory beside it.

To open a particular workspace from PowerShell:

```powershell
.\Patch.exe 'C:\Users\you\code\project'
```

No installer, Go installation, or separate Ghostty installation is needed to run
the package. Settings and workspace state use `%APPDATA%\Patch` by default;
`PATCH_HOME` overrides that location. The package is unsigned.

## Keyboard shortcuts

| Action | Windows shortcut |
|---|---|
| Open folder | `Ctrl+Shift+O` |
| New workspace / workspace tab | `Ctrl+Shift+N` / `Ctrl+Shift+T` |
| Split right | `Ctrl+Shift+D` |
| Split down | `Ctrl+Shift+E` |
| Close panel | `Ctrl+Shift+W` |
| Close workspace | `Ctrl+Shift+Q` |
| Previous / next workspace | `Alt+Left` / `Alt+Right` |
| Move workspace left / right | `Alt+Shift+Left` / `Alt+Shift+Right` |
| Hide / show sidebar (sidebar layouts) | `Ctrl+Shift+B` |
| Settings | `Ctrl+Shift+,` |
| Refresh active panel | `Ctrl+Shift+R` |

Standard editing commands follow the focused tool. Bindings are defined in
[`platform.go`](../internal/app/platform.go) and [`commands.go`](../internal/app/commands.go).

## Build

Install the Go version declared in `go.mod`. From the repository root, on
Windows, macOS, or Linux:

```sh
go run ./scripts/build-windows.go
```

With Make installed, `make build-windows` runs the same command. It invokes the
repository-pinned MyGo CLI:

```sh
go tool mygo build -platform windows/amd64 -skip-installer -o dist
```

MyGo downloads and SHA-256-verifies the pinned terminal plugin's `ghostty-vt.dll`,
copies runtime resources, embeds Windows version information and a per-monitor
DPI manifest, and builds a GUI executable without a console window. The font is
embedded in the executable; its license and provenance files ship in `fonts/`.
The build uses Go's cross-compiler and requires no C compiler or NSIS.

The helper checks the executable format and required resources, then packages
the **whole** `dist/windows-amd64/` directory into
`dist/patch-0.1.0-windows-amd64.zip`. The version comes from `mygo.json`.
`dist/windows-amd64/` is generated output: MyGo builds in a fresh staging
directory and replaces that entire target on success, removing stale resources.
Keep personal files elsewhere. The ZIP is also written to a temporary file before
replacing the previous archive.
MyGo automatically embeds `resources/icon.png` when supplied; Patch currently
has no application icon asset.

The Actions workflow builds the same package on a native Windows runner and can
also be started manually. Building and checking the archive do not exercise the
Windows UI: native launch, DirectWrite font rendering, terminal input, and window
controls require a Windows runtime check.

References: [MyGo distribution](https://github.com/egoist/mygo/blob/6842ad5e0aa2/docs/distribution.md),
[terminal plugin](https://github.com/egoist/mygo/blob/6842ad5e0aa2/docs/plugins/terminal.md).
