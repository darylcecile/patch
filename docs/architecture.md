# Native shell architecture

## Boundaries

```text
main.go                      CLI root, PATCH_HOME
  internal/app               native window/chrome, menus, drag/resize, settings
    internal/workspace       durable descriptors + split/tab operations + geometry
    internal/config          versioned settings and safe JSON storage
    internal/panels          native factory registry + live panel resources
      mygo/ui                native controls, tree, code editor, GPU renderer
      mygo/plugins/terminal  PTY + libghostty-vt terminal emulator
      mygo.WebView           optional, explicitly opened Browser page
```

`main.go` resolves an explicit workspace root without changing the process working directory. App startup shows the native window first, then reads settings/session and starts panel I/O in the background. An explicit project root selects an existing matching workspace or creates one. Without a root or a saved session, the native welcome screen asks the user to open a project folder; Finder's working directory is never treated as a project. Invalid existing sessions produce a visible error and preserve the original file. A new project receives a left stack of terminal/changes and full-height files on the right.

## Durable model, live resources

`internal/workspace` contains no UI or process objects. Its binary split tree has stable split/group IDs; leaves retain ordered stable-ID panel descriptors and an active panel for compatibility with saved grouped panels. Model methods own moves, edge splits, center swaps, reorder/close, focus, zoom, ratio validation and geometry. New UI panels open as splits, and a single header selector exposes legacy grouped panels. The core retains center-merge support for its existing data model, but the current shell's center drop calls `SwapPanels`. The renderer does not derive a second layout model.

`internal/panels.Registry` maps a descriptor's kind to a native `Factory` and its stable ID to an `Instance` with `View(*ui.Context)` and `Close()`. `Sync` runs after model operations. Existing instances survive; new descriptors create instances; missing IDs dispose their resources. Unknown kinds show an unavailable surface without deleting descriptors. This is the present native extension seam, not a dynamically loaded extension runtime.

An optional `Menu(*ui.Menu)` contributes panel-specific view commands to the host's header context/overflow menu. Changes uses this for unified/split and collapse/expand actions, keeping a single host header. Keyboard users can focus that header and open the same native popup with Shift+F10; changing the active group does not steal focus from its header controls.

Panel content containers are flat, stable-ID children of the window root. Their absolute rectangles come from `Workspace.Layout`. Group cards/headers render separately. Docking therefore changes panel geometry without changing widget parent ancestry, an important distinction in mygo's identity system. Inactive panel content remains `Hidden`, preserving widget state. Persistent `ui.Handle`s explicitly restore content focus when changing active panels, and pending focus requests are canceled when selection changes. No `ui.Element` is retained across builds.

Terminal creation and native-library loading run on a worker goroutine. Its publication/lifecycle lock ensures that a terminal finishing startup after close is disposed instead of leaked. Closing a panel or application calls `Terminal.Close`; navigation and resizing never do. Native shell processes and scrollback are session-local and do not survive application quit.

## Input and layout

- Header `Drag` values carry workspace and panel IDs. Mygo applies its movement threshold and Escape cancellation. The current target's local pointer chooses an edge or center; preview identity and docking zone are calculated together each frame. Center swaps positions, while edges move/split. Input-preserving chrome and an explicit post-move focus request return typing to the moved panel.
- Drag/drop queries collect model actions and apply them after view construction. A preview is painted above all content and ignores input throughout its subtree, so its label cannot intercept release.
- Split dividers use a 12-point invisible hit region and resize cursor, with a short grip only for keyboard focus. Pointer capture maps absolute pointer displacement from the press anchor directly to the model ratio. The stored initial size is not replaced by a clamped visual size on each event. The sidebar gesture starts from its actual rendered width. Arrow keys move the ratio, Enter/double-click reset it, and menus equalize all splits.
- The core keeps saved ratios when the viewport temporarily compresses panes. Narrow cards keep a full actions menu, titles/paths truncate and Above-mode tabs scroll. View → Customize View → Tab Style chooses Above/Left Side/Right Side. Sidebar visibility never changes that placement, and viewport size never switches modes. Version-1 config overlays `sidebarPosition` with a default of left; side, visibility and saved width remain independent.
- A left expanded sidebar uses compact Kero-style chrome when native caption controls fit on its left. Right-side/collapsed layouts use a full header; native right-caption clearance (including Windows' controls) also requires that header and places panels below it. Sidebar resizing anchors to the rendered edge, reversing horizontal movement for the right sidebar.
- Every titlebar/panel action has a label. Native menus provide alternatives to dragging. Only the titlebar's background drags the window; panel grips drag panels.

## Background I/O and persistence

Mutable app/core state belongs to the main thread. File listings, file previews, Git subprocesses, terminal startup and session loading run outside render. Results enter via `Window.Update`. File previews check regular-file metadata before opening, then read UTF-8 text bounded to 512 KiB. Directory expansion is lazy; a listing displays at most 1,000 entries and recursion is bounded. Git commands have a 10-second deadline, bounded stdout previews and separately bounded error diagnostics. Oversized diffs remain readable truncated previews. `GIT_OPTIONAL_LOCKS=0` prevents optional Git index writes. Loading/error/empty states retain a focus target, and pending panel focus is not overwritten by the previously focused content.

The Changes owner builds immutable staged/unstaged/untracked snapshots into native `editor.DiffView` file cards. A virtual list supplies sticky collapsible file headers; unified/split presentation uses the same diff documents. Files and Changes each composite their nested rectangular clips inside a viewport-sized cache boundary, so their outer GPU quad remains subject to the panel's rounded ancestor clip. This is deliberately scoped to those native subtrees; Browser panels are not cached. CPU headless clipping does not prove the Metal result.

The app's persistence writer receives a serialized, detached workspace snapshot and a copied config. A single worker coalesces snapshots for 250 ms, writes through the core's validated atomic-save functions and flushes pending state at shutdown. Divider movement updates only the in-memory ratio; release persists it. Write errors return through a bounded channel and remain visible in the window. Corrupt or unsupported originals are not replaced by ordinary saves. `PATCH_HOME` also sets mygo's user-data path, isolating window geometry.

## Optional web content

The default window uses `WindowOptions.Content`, never a main web page. A Browser panel initially shows native URL controls. Submitting HTTP(S) creates one retained system webview with `PageOptions.Untrusted: true`; native URL/load-error controls remain outside it. Accepted URLs and committed navigation/redirect URLs pass through `Host.SetResource` to the main-thread `Workspace.SetPanelResource` command and the debounced writer. Typing and invalid addresses do not alter the descriptor. Restart restores the address without starting a webview until **Go** is invoked. No Go service is bound to the page. Destroying the panel destroys the webview. Browser navigation is independent from terminal/editor resources. Native window captures do not include system-webview pixels.

The extension PRD describes a future TypeScript-facing host and capability APIs. Neither that runtime nor hot reload is present here; the current registry establishes lifecycle/identity seams without choosing a JavaScript engine prematurely.

## Design and reused components

The light palette is a vertical lavender-to-warm-cream backdrop with borderless near-white cards, 12-point outer/11-point content corners, plain 9-point gaps, single-line 36–40-point card headers and 13-point native system text. Toolbar icons have no resting boxes and appear only for the active/hovered/keyboard-focused card; wide cards show the four reference actions, and narrow cards show an overflow menu. Left-side navigation is the default. Above mode dedicates its titlebar to workspace pills and utility controls, without a machine block. Pill/select/close controls use explicit vertical alignment and icon sizes that fit their actual content boxes. The Files splitter retains mygo's behavior with a scoped transparent divider-paint theme, while its child controls retain the ordinary content theme and focus treatment.

Monospaced terminal text uses Menlo on macOS and the system monospace font elsewhere, with a process-local JetBrainsMono Nerd Font Regular fallback registered once before terminal creation. Menlo preserves correct BMP fallback and primary ASCII/bold/italic selection. The replacement fallback's advance fits the terminal cell and avoids the 8/13 ink shrink caused by the former symbols-only Mono asset, without enlarging the primary font or grid. Actual overhang/spacing and Windows DirectWrite behavior require native observation. See `resources/fonts/PROVENANCE.md` for the pinned unchanged asset and constituent licenses. Shell selection remains mygo's; the asynchronous header label mirrors its platform defaults rather than assuming zsh on Windows.

Visual/behavior references: [Kero](https://github.com/egoist/kero/tree/fd335a5551ce4e2d2d14e1e7d257b37214097cf4) for grouped sidebar/card/backdrop relationships; [GoRex](https://github.com/egoist/gorex/tree/b206e67bf0ac422afebc5c9e701a38dd67f2c837) for workspace pills and panel toolbar organization. No GoRex implementation code is copied. [Apple split views](https://developer.apple.com/design/human-interface-guidelines/split-views) informs visible dividers, nested axes and resize affordances.

Reused mygo primitives include Row/Column, Button/Icon, Tree/TreeItem, Scroll/ScrollHorizontal, SplitPercent (inside the Files panel), Segmented, Switch, Checkbox, Slider, TextInput, DialogBase, Menu/context menus, editor.View and terminal.View. Small toolbar glyphs are original SVG paths rendered through `ui.Icon`. Chrome is not a custom-canvas control system.

## Verification boundary

Pure core tests verify model/persistence/diff transitions. Focused `ui.Tester` checks exercise thresholded drag preview/cancel/center swap and restored focus, pointer resize/reset, independent placement/visibility, right-side resizing and narrow access to panel menus. `go vet` and `go tool mygo vet` cover Go and build-value lifetimes. Native build evidence establishes packaging, not visual or interactive acceptance. Running-window verification must additionally exercise the real PTY, GPU clipping after moves/scrolling, menu checkmarks, hover/pressed/focus alignment, glyph ink/grid alignment, the diff surface and restart persistence. The debug-only `e2e` build plus explicit `MYGO_AUTOMATION` socket enables that inspection without a production listener. Windows build/runtime boundaries are documented separately in `docs/windows.md`.
