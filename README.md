# Patch

A native, panel-based development workspace built in Go with [mygo](https://github.com/egoist/mygo). The window chrome, terminal, file tree, previews, Git changes and settings are GPU-rendered native UI. There is no web frontend or JavaScript runtime in the shell.

## Run

The terminal requires macOS 13 or later, or Windows 10 version 1809 or later. Native development verification currently runs on Apple Silicon macOS; see [Windows builds and support status](docs/windows.md) for the portable x64 package. Go automatically downloads the pinned **Go 1.27.3** toolchain when `GOTOOLCHAIN=auto` (the default); no global Go replacement is needed.

```sh
go run . /absolute/path/to/project
# Or use the current directory explicitly:
make run ARGS="$PWD"
# Without a root, restore the previous session or choose a project folder:
go run .
```

The first development launch downloads mygo's checksummed `libghostty-vt` library into its cache. This happens in the background; the window opens while the terminal starts. Packaged apps include the library.

```sh
make build
open dist/darwin-arm64/Patch.app
# To pass a root and isolated state directly to the bundle:
PATCH_HOME="$PWD/.groundwork/tmp/patch-state" \
  dist/darwin-arm64/Patch.app/Contents/MacOS/Patch "$PWD"
```

`make build` produces an ad-hoc-signed local `.app` with `go tool mygo build -skip-dmg`. Distribution signing/notarization is not configured. The framework and CLI are pinned together at `6842ad5e0aa2cd168166e0a4ae12c0cd0f527a3d`.

## Working in Patch

Choose a project with **Open folder…** or pass its directory on the command line. The first project arrangement places a real local shell above Git changes on the left, with project files on the right. A fresh Finder launch starts with a folder-led welcome screen. New workspaces point to existing directories; Patch does not initialize repositories or run project commands automatically.

- **Arrange panels:** drag a header. A visible preview shows left, right, top, bottom or center docking. Center swaps the two panels in place. Release commits; **Escape** cancels. Panel moves stay within their workspace; opening a new panel creates a split.
- **Resize:** drag the plain 9-point card gaps, with a 12-point invisible hit region; the pointer changes to a resize cursor. Double-click to reset a split; focus a divider and use arrows to resize or Enter to reset. A short grip appears for keyboard focus. Panel menus also equalize all splits.
- **Panel controls:** a single-line header holds the title and path. Split right/down, zoom/restore and close actions appear on the active, hovered or keyboard-focused card. Narrow cards use **Panel actions** (`…`); a header context menu and the native Panel menu expose the remaining actions. Older saved tab groups retain every panel in the header's **Switch panel** selector.
- **Workspaces:** choose **Above**, **Left Side** or **Right Side** from **View → Customize View → Tab Style**. Above uses horizontally scrolling workspace pills; side placements use a sidebar. Drag to reorder; context menus offer earlier/later/close. Hide/Show Sidebar changes only visibility within a side placement. Resizing the window never switches the chosen mode; a narrow sidebar clamps its displayed width while retaining its saved width. Filesystem-root workspaces remain available with readable root labels.
- **Project files:** browse directories and select a UTF-8 text file for a native, syntax-highlighted read-only preview. File reads are limited to 512 KiB; binary and larger files show an explicit explanation. **Panel → Refresh** reloads the root listing.
- **Changes:** staged, unstaged and untracked changes appear as collapsible file cards with native syntax-colored diffs and word highlights. Right-click the panel header, or focus it and press Shift+F10, for Unified/Split and collapse/expand commands; narrow headers expose the same actions through overflow. Narrow panels and new/deleted files use unified presentation. File headers remain visible while scrolling. The panel shows clean, binary, oversized and error states explicitly. Workspaces opened in a subfolder retain repository-wide, root-relative context. Refresh reloads from disk; Git runs with `GIT_OPTIONAL_LOCKS=0`, avoiding optional index writes.
- **Native controls:** an opt-in gallery of reusable mygo controls under **Developer → Native Controls**, limited to one newly opened gallery per workspace. Existing duplicate panels in older sessions are preserved.
- **Browser:** opt-in from Workspace or a panel menu. A system webview is created only after submitting a complete HTTP(S) URL. It has URL/back/reload controls and visible load errors. Accepted URLs and committed redirects persist in the panel descriptor; restarting restores the address, and **Go** reopens the page on demand. It has no privileged Patch API bridge.

Terminal processes and editor/widget instances stay alive across docking, resizing and workspace/tab navigation. Closing a terminal or workspace ends its shell. Quitting Patch ends local shells; restarting restores the panel layout and starts new shells, rather than restoring processes or scrollback.

### Keyboard and native menus

These are the macOS shortcuts. Windows uses distinct app shortcuts that leave ordinary Control terminal keys available; see [Windows](docs/windows.md). A packaged Windows build is not evidence of a native Windows runtime check.

| Shortcut | Action |
| --- | --- |
| `⌘N` | New workspace |
| `⌘T` | New workspace tab |
| `⌘O` | Open project folder |
| `⌘D` / `⌘⇧D` | Split right / down |
| `⌘⇧Enter` | Zoom / restore active group |
| `⌘⇧0` | Equalize splits |
| `⌘W` / `⌘⇧W` | Close panel / workspace |
| `Ctrl+Tab` / `Ctrl+Shift+Tab` | Next / previous panel in a legacy group's selector |
| `⌘⌥Right` / `⌘⌥Left` | Next / previous workspace |
| `⌘⌥Down` | Focus next panel group |
| `⌘⌥M` | Swap active panel with the next panel |
| `⌘R` | Refresh active files/changes panel |
| `⌘B` | Hide/show the sidebar in a side-placement mode; no effect in Above mode |
| `⌘,` | Settings |

The Workspace, Panel and View menus expose the same actions. Standard native Edit menu copy/paste actions work with terminal and text controls.

## Settings and saved state

Use **View → Customize View → Tab Style** for **Above / Left Side / Right Side**, with a checkmark on the current placement. **View → Hide Sidebar / Show Sidebar** is separate and enabled only for side placements. Open **View → Settings** for System/Light/Dark appearance, Comfortable/Compact density, accent and sidebar sizing. Mode, sidebar position and visibility save independently. Legacy `"both"` navigation loads as `"sidebar"`; older version-1 files default the new sidebar position to `"left"`.

State normally lives in `~/Library/Application Support/Patch` on macOS. Set `PATCH_HOME` to use an isolated directory; this also isolates mygo's window state. `config.json` stores preferences, `session.json` stores workspace/layout descriptors, and `window-state.json` stores window geometry.

Example `config.json` (edit while Patch is closed):

```json
{
  "version": 1,
  "sidebarVisible": true,
  "sidebarWidth": 240,
  "sidebarPosition": "left",
  "navigation": "sidebar",
  "appearance": "light",
  "density": "comfortable",
  "accent": ""
}
```

Accent tokens are `""` (lavender), `"blue"` and `"green"`. Sidebar width is clamped to 180–420 points. Invalid/unsupported saved documents surface an error and are preserved; autosave does not overwrite them. To recover deliberately, close Patch and move the affected JSON aside before restarting.

## Development and verification

```sh
make check                  # Go tests, vet and native executable build
go tool mygo vet             # mygo build-value lifetime checks
make debug                  # build/patch with opt-in native automation
PATCH_HOME="$PWD/.groundwork/tmp/native-state" \
MYGO_AUTOMATION="$PWD/.groundwork/tmp/patch.sock" \
MYGO_FRAME_STATS=all \
  ./build/patch "$PWD"
```

The automation build accepts the [mygo NDJSON socket protocol](https://github.com/egoist/mygo/blob/6842ad5e0aa2cd168166e0a4ae12c0cd0f527a3d/docs/plugins/automation.md). The socket is disabled unless `MYGO_AUTOMATION` is set and is absent from ordinary release builds. Ensure its parent directory exists. `MYGO_FRAME_STATS=all` logs actual native frame phases; it is a development diagnostic, not a frame-rate guarantee.

Go tests include pure workspace/persistence/diff checks and focused mygo `ui.Tester` interaction checks. Headless results establish view behavior, not native GPU clipping, font rendering or PTY continuity; those require the running-window evidence collected separately.

Useful semantic keys: `open-project`, `browse-project`, `toggle-sidebar`, `open-settings`, `new-workspace`, `new-workspace-tab`, `workspace-name`, `workspace-root`, `appearance-setting`, `density-setting`, `accent-setting`, `sidebar-width-setting`. Placement is selected through the native menu path **View → Customize View → Tab Style**. Repeated controls have keys `panel-grip`, `panel-selector`, `split-right`, `split-down`, `zoom-panel`, `close-panel`, `panel-menu`, `terminal-content`; select their unique runtime ID from the tree. Divider keys are `divider-<stable split ID>`. Terminal content is labelled **Terminal input**; its header shows the selected shell executable and reported working directory. Native captures of an embedded Browser contain a hole at the system webview; use an OS screenshot or a separate webview capture for that region.

Terminal text uses **Menlo on macOS** and the system monospace font elsewhere, with **JetBrainsMono Nerd Font Regular as a fallback** for supported prompt icons. Menlo remains the primary macOS font, including ASCII bold/italic; terminal size and grid are not enlarged to size icons. The fallback is registered only within Patch; shell configuration and system fonts are not changed. See [asset provenance and licenses](resources/fonts/PROVENANCE.md).

Kero-style larger icon sizing is intentional: wide icon ink can overhang neighboring cells and overlap tightly adjacent non-space characters. The existing spaced prompt keeps the larger icons without a font or prompt rewrite.

## Architecture and roadmap

- [Architecture](docs/architecture.md): core, rendering, resource lifecycle and persistence boundaries.
- [Native verification and screenshots](docs/verification.md)
- [Product and extension proposal](docs/PRD.md): native extension platform, reusable APIs and the proposed TypeScript SDK/hot reload workflow.
- [Research](docs/research.md): framework and design references, trade-offs and provenance.

The current extension seam is a Go panel-factory registry. A JavaScript extension runtime, TypeScript SDK, extension permissions, hot reload and cross-window panel transfer are proposed work in the PRD; they are not implemented by this initial shell.
