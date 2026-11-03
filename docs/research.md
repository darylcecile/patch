# Patch: extension and native-UI research

**Research date:** 2026-11-03. **Status:** source research with a separately attributed M0 runtime-evidence addendum; not a runtime benchmark. Product decisions and acceptance criteria are in [PRD.md](PRD.md).

## 1. Recommendation

Build Patch's shell and interaction-critical components in Go with **MyGo's custom GPU-rendered UI**. Let extensions supply commands, asynchronous service providers, data and typed native component descriptions. **Choose the language independently from who owns the hot path:** layout, hit testing, pointer capture, scrolling, selection, text composition, focus and rendering belong to the host, regardless of whether extension logic is JavaScript, Lua, WASM or a compiled executable.

Recommend **TypeScript compiled to JavaScript as the primary authoring experience**, with a small Go helper process running **goja as the first runtime candidate**. This is a provisional implementation choice, not a performance conclusion. A bounded prototype must compare it with a Node helper and QuickJS on the same representative extensions. Ship one supported runtime initially. Keep a language-neutral protocol so an expensive parser or existing Go tool can become a sidecar later without changing the UI contract.

The evidence supporting this separation is stronger than the evidence supporting any particular runtime:

- Raycast documents React/TypeScript descriptions reconciled to native AppKit views, across a Node-process RPC boundary. This establishes a working product pattern; it does not establish Patch's performance. [RAY-1, RAY-2]
- Blender is the closest combined product precedent: Python panels use host `UILayout` controls inside a user-reconfigurable area/editor layout. Its explicit warnings about expensive draw callbacks show why Patch should adopt the host component model while moving extension execution off the draw path. [BLEND-1–4]
- VS Code makes contributions discoverable before activation and withholds access to its own DOM. This separates supported product contracts from renderer internals. [VSC-1–4]
- Neovim explicitly restricts editor API calls from fast/libuv contexts. Interpreted code still needs scheduling discipline. [NVIM-1, NVIM-2]
- Qt recommends asynchronous work and short bindings even with a native renderer and an integrated JavaScript engine. [QT-1]
- MyGo's view runs on the main thread and may build more than once in a frame. Calling an extension synchronously from a view would make extension work part of input/render latency. [MYGO-2]

**Current delivery is research plus an initial shell.** The public runtime/SDK, permissions and hot reload remain future proposals. A singleton native-control gallery is a developer reference, not an extension runtime. Following the M0 baseline's **Opus 5.5 xhigh visual PASS**, the latest navigation, font, pill-state and native-diff refinements also received scoped [native acceptance](verification.md#refinement-batch-tab-style-diff-review-glyph-size). This does not establish proposed extension performance or native Windows behavior.

## 2. Evidence and revision policy

Labels used here:

- **Documented/source:** an upstream document or inspected source says this; no local execution implied.
- **Upstream measurement:** published with an upstream fixture; not reproduced here.
- **Inference/recommendation:** a conclusion for Patch from those sources.
- **Target:** desired Patch behavior awaiting implementation and measurement.
- **Unknown:** evidence not collected or a decision the prototype must resolve.

Context7 was used first to discover and query current APIs for MyGo, VS Code, Neovim, Zed, Emacs Lisp, Sublime, IntelliJ, Raycast, Blender, remote-dom, Qt, goja, QuickJS, Node.js, wazero, GopherLua and Go. Consequential claims below were checked against primary project documentation/source, rather than relying on generated API summaries. Source links in the register were accessed on **2026-11-03**, except the explicitly coordinator-supplied Apple HIG and Godiff references. Most projects publish rolling documentation; those pages are not immutable release guarantees.

MyGo was inspected at commit [`6842ad5e0aa2cd168166e0a4ae12c0cd0f527a3d`](https://github.com/egoist/mygo/commit/6842ad5e0aa2cd168166e0a4ae12c0cd0f527a3d), dated 2026-11-03, and its sources below are pinned to that revision. Patch's `go.mod` pins it as `v0.0.0-20261103171621-6842ad5e0aa2`, with Go 1.27.3. The earlier reported local Go 1.24.2 is historical setup context, not a current blocker. **The coordinator's [runtime report](verification.md) now records debug/release builds and scoped native acceptance**, separately from this source research. This documentation task did not execute the toolchain or application.

### M0 runtime evidence supplied by the coordinator

The [verification report](verification.md) owns build-specific Apple M2/macOS 26.6.2 observations and hashes. Earlier native checks established docking/swapping, resizing, immediate typing, terminal PID/TTY continuity, restart, folder selection, friendly non-Git state and legacy-panel retention. Invalid saved-file handling has core-test evidence, not a native corruption journey. The [completed refinement section](verification.md#refinement-batch-tab-style-diff-review-glyph-size) records final debug `a1fa5260…`, release `50161696…` and the builds supporting each latest observation.

The earlier one-navigation shell received Opus 5.5 xhigh visual PASS. The [latest native PASS](verification.md#verified-natively) confirms **View → Customize View → Tab Style → Above/Left/Right**, independently persisted mode/`sidebarPosition`/visibility, sidebar-only collapse/expand, no toggle Above and no mode change at 900×700. Hostname removal, aligned hover-close X, whole-pill focus/pressed states and mirrored Right Side inset are verified.

Current terminal fallback is **non-Mono JetBrains Regular secondary to Menlo on macOS or the system primary on Windows**, replacing Symbols Mono sizing. Native prompt sizing passed with the user's accepted Kero-like ink overhang beside non-space text; this is not Windows font verification. Files GPU-composited corners and Changes corners passed. Native `editor.DiffView` now has verified readable EOF rows, file cards, changed-line/word colors and context after fixes for the highlighter crash, row height and phantom newline rows. Popup Unified/Split/Collapse all/Expand all actions remain source/headless-checked because native tooling could not reach them. [Refinement results and limits](verification.md#refinement-batch-tab-style-diff-review-glyph-size)

**Scope limits:** native runtime verification is macOS Apple silicon only. Windows x64 portable output has been built and its PE/DLL/archive inspected; Windows runtime remains unverified. A native-Windows GitHub Actions build is configured for the forthcoming `main` push, not yet a passed CI result; see [Windows packaging](windows.md). Screen-recording permission prevented OS-composited traffic-light/webview pixel capture; page beacons verify browser lifecycle instead. Earlier captures are 1440×814 pt at 2×, not a full viewport/platform matrix. One **96,240 KiB (~94 MiB) main-process RSS snapshot** excludes children/WebKit/GPU footprint and is not a budget pass. Startup/input/idle-CPU and extension-runtime budgets remain unmeasured. These are attributed coordinator findings, not tests run by this research author.

## 3. MyGo: what “native” means and what Patch must own

### Rendering and platform contract

**Documented:** MyGo offers two surfaces: a system webview, or `ui.View` built in Go and drawn by MyGo. The latter uses Metal on macOS, Direct3D 11 on Windows and OpenGL on Linux, with CPU rendering also described in its architecture. Platform windows, menus, text services and accessibility bridges coexist with custom controls; “native UI” does **not** mean every button is an AppKit/Win32/GTK widget. Native-only windows do not need to start a webview. [MYGO-1, MYGO-7]

MyGo documents `CGO_ENABLED=0` support through runtime native interop, not an absence of native dependencies. Its native Linux surface still requires GTK 3; a webview adds WebKitGTK requirements. The inspected README requires Go **1.27.3+**, while some guides say 1.27+. Use the actual pinned module/toolchain requirements. Go's automatic toolchain selection can download a newer toolchain with `GOTOOLCHAIN=auto`; a machine having `go` installed does not prove it can build the pinned dependency offline. [MYGO-1, GO-2]

**Inference:** the toolkit fits the requested Go/native direction, but its broad API and active changes make version pinning and a small adapter prudent. A framework source claim is insufficient evidence of Patch's IME, screen-reader, docking or rendering quality across platforms.

### Build lifetime, identity and threading

**Documented:** view functions execute on the main thread; one frame can contain multiple builds. `ui.Element` is a build-scoped value, while `ui.Handle`, `ui.Services` and app-owned widget state are persistent. `Window.Update` moves model changes onto the main thread and requests a frame. `Invalidate` alone does not synchronize unsafely shared state. Keys are scoped to their parent: changing a keyed parent changes descendant identity. [MYGO-2]

**Patch implication:** retain extension trees and Go widget state by stable pane/node IDs; never retain `ui.Element` in a worker or extension object. Decode/validate messages away from rendering, then publish a bounded update on the UI thread. Moving a pane between split-tree parents requires explicit state/focus preservation: merely giving it the same leaf key cannot guarantee retention under MyGo's parent-derived identity.

### Existing primitives versus application layout

MyGo already documents keyboard-resizable splitters, percent-based split sizing, tab keyboard behavior, a sidebar, list/table/tree/outline controls, native code editing and a native terminal plugin. These are appropriate building blocks; they do not constitute a general panel docking/workspace manager. `ui.Split` has its own minimum-space behavior, so Patch must account for panel minima instead of assuming a split API implements its layout policy. [MYGO-3, MYGO-4, MYGO-5, MYGO-8]

The terminal plugin starts a real program in a PTY through `terminal.New`, renders with libghostty-vt and closes explicitly. The native library is loaded at runtime and bundled by `mygo dev`/`mygo build`: **no cgo is not native-library-free**. Keeping the same terminal object across layout changes is distinct from implementing a daemon that survives application exit. [MYGO-8]

Lists build visible rows plus an overscan region, estimate variable heights, and can anchor scroll/selection to item keys. Focused rows may remain built outside the visible region. [MYGO-5] The upstream “millions of rows as fast as ten” wording is a broad claim: a Patch extension that serializes a million UI rows would defeat virtualization before MyGo sees them. A native virtual list therefore needs a **range-based data protocol**, not just a virtualized Go renderer.

### Embedded webviews

`Window.NewWebView` creates the system page; `ui.WebView` places it under the native surface. A hidden/unbuilt view keeps its page by default. `Destroy` is explicit. MyGo documents focus entering/leaving a page, native overlays, clipping, and untrusted-page configuration. It also documents limitations: host capture shows a hole rather than the page; a separate webview capture is needed; effects sampling host pixels cannot sample the page; Windows accessibility has a distinct page-window structure. [MYGO-6]

**Patch implication:** creation must be opt-in and lazy. Hiding a pane is not equivalent to releasing browser memory. Page lifetime and privileged service exposure need explicit policy. Third-party extension HTML must not inherit the app's trusted-page bridge. An optional webview can be a useful rich-document escape hatch while shell navigation and native extension panels remain native.

### Performance evidence is scoped

The pinned performance page reports CPU `ui.Tester` steady frames on Apple M5/macOS arm64/Go 1.27.1, with `CGO_ENABLED=0 GOMAXPROCS=1`, using medians of six warmed 300 ms samples. It reports 397 µs for 1,000 labels and 14.5 µs for visible rows of a million-row list. A separate retained-heap fixture reports approximately 2.50 MB and explicitly excludes native-window process memory and GPU resources. The primary fixture date is 2026-10-08; Markdown fixtures have a separate 2026-10-14 date. [MYGO-9]

These are useful **upstream measurements**, not end-to-end GPU/input measurements, whole-app footprint figures, cross-framework comparisons or Patch results. This task ran no Patch benchmarks and establishes no performance win over Electron, VS Code, Zed or another terminal.

## 4. Reference products: lessons and limits

| System | Documented architecture / authoring | What Patch should take | What does not transfer automatically |
|---|---|---|---|
| **VS Code** | JSON contributions, lazy activation, Node local/remote hosts or browser workers; no direct workbench DOM access. | Discover commands/panels/settings without running code; explicit service boundaries and supported contribution slots. | VS Code is not a native-renderer precedent. A separate host is not a capability sandbox. Its API breadth is decades of product work. [VSC-1–4] |
| **Neovim** | Lua 5.1 is the permanent Lua interface; LuaJIT is recommended where supported. Lua, RPC and external plugins access a common editor API. | A scriptable core, small composable commands, stable API metadata, first-class headless/remote control. | In-process Lua does not prevent UI starvation; libuv and fast-event callbacks have editor-state restrictions. [NVIM-1, NVIM-2] |
| **Zed** | Declarative resources plus procedural Rust compiled to WASM; current development docs use `wasm32-wasip2`. | A small versioned host import surface, portable packages, explicit capability methods. | The inspected documented contributions are languages, debuggers, themes/icon themes, snippets and MCP servers—not arbitrary GPUI panel rendering. WASM alone does not create a UI API. [ZED-1, ZED-2] |
| **Emacs** | Lisp definitions can be evaluated into the running editor; commands, modes, hooks and customization share a programmable environment. Lisp threads share memory and are mostly cooperative. | Immediate feedback, inspectable commands/settings, a REPL and small local customizations. | Live redefinition does not guarantee cleanup, isolation or rollback; Lisp threads are not general parallel execution of editor state. [EMACS-1–3] |
| **Sublime Text** | Python runs in a separate `plugin_host`; lifecycle callbacks and asynchronous event variants are explicit. | A native editor can expose a useful interpreted API without moving its renderer into Python. Host failure should leave user work savable. | Plugin processes/environments can be shared. Synchronous callbacks still matter. Runtime compatibility must be pinned, not assumed. [SUB-1, SUB-2] |
| **IntelliJ Platform** | Swing EDT plus background tasks, read/write state access rules, services and disposables. Dynamic unload has restrictions. | Scoped ownership, disposal and cancellation; stable IDs rather than long-lived pointers into changing host state. | “Hot reload” is not simply reloading a module: retained class loaders, native libraries and non-dynamic extensions can require restart. [IDEA-1–3] |
| **Raycast** | React descriptions transformed to native AppKit UI; a child Node process, worker isolates, defined RPC APIs. | Strongest directly documented example of familiar TS UI authoring with host-owned native controls. | Its official security page says file/network APIs are not further sandboxed. Its constrained launcher views are not a complete docking IDE model. [RAY-1–3] |
| **Blender** | Registered Python `Panel`/`UILayout` controls, operators/properties, custom list rows, reconfigurable editor areas. | Closest match for script-authored native panels within a configurable workspace; reuse semantic controls and host interaction. | Python draw callbacks can still stall UI; Python integration is not generally thread-safe. Adding a panel to an editor is not automatically defining a new arbitrary editor type. [BLEND-1–5] |
| **Shopify remote-dom** | Remote mutations, node IDs, event/method bridging, receiver-side rendering, optional batching and retain/release hooks. | A narrow tree/update protocol and explicit callback lifetimes; separate rendering from remote authoring. | The supplied implementation assumes JS and DOM-like environments; a Go/MyGo renderer and permission model would be new work. Do not describe it as an existing Go bridge. [REMOTE-1, REMOTE-2] |
| **Qt/QML** | Declarative object trees/property bindings, JavaScript logic, native integration and a scene graph. | Declarative does not mean HTML; expose semantic properties and let the host render. Keep bindings cheap and profile real workloads. | It would introduce another UI engine/toolchain rather than layer naturally onto MyGo. QML is not a generic no-cost scripting adapter. [QT-1, QT-2] |

### VS Code: contributions, activation and webview discipline

The manifest describes commands, keybindings, menus, configuration, views, colors and other slots without requiring their implementation to run. Activation can follow a command, language, view or workspace condition. Since VS Code 1.74, several contributed features infer their activation, avoiding duplicate declarations. Patch should similarly infer activation for its own contributed commands/panels and reserve explicit activation for background requirements. [VSC-2, VSC-3]

The host guide distinguishes local Node, browser-worker and remote Node execution. `extensionKind` expresses preference for workspace/UI location; it is not an authorization rule. Patch can retain URI-based resources and explicit workspace handles without committing to a remote-host product in the first shell. [VSC-1]

The webview guide explicitly calls webviews resource-heavy and recommends them when normal APIs are inadequate. It recommends serialized state over retaining hidden contexts and documents CSP, content roots and messaging. Patch's opt-in surface follows that product trade-off, but MyGo has different default page-retention and capture behavior. [VSC-5, MYGO-6]

### Blender: the closest panel-and-workspace precedent

Blender's `Panel` API registers a class with a stable `bl_idname`, a label and a location expressed by editor space, region and context. Its `draw(context)` builds `self.layout`; the layout supplies rows, columns, property editors, labels and operator buttons. `poll` expresses contextual availability. This is a useful semantic model: an extension describes an action or a property, and the application supplies interaction, appearance and integration. It is not HTML, and it does not require an extension to implement pointer hit testing or a text editor. [BLEND-1, BLEND-2]

The following is the shape of Blender's **existing API**, not Patch syntax:

```python
class ExamplePanel(bpy.types.Panel):
    bl_idname = "OBJECT_PT_example"
    bl_label = "Example"
    bl_space_type = 'PROPERTIES'
    bl_region_type = 'WINDOW'
    bl_context = "object"

    def draw(self, context):
        self.layout.prop(context.object, "name")
        self.layout.operator("object.select_all")
```

Blender's area model independently supports resizing, splitting, joining, swapping content and maximizing/restoring an area; its documented docking operation can be canceled before completion. [BLEND-4] That independence is directly relevant to Patch: **pane placement is host state, panel content is extension state**. Changing placement should not require an extension to rebuild its business model, and each editor/pane can use the same command/property vocabulary.

The most instructive limit is in Blender's own `UIList` documentation: `draw_item` is called for visible collection items, while an advanced example explicitly warns that iterating all vertices in a draw function is a very bad idea for UI performance. [BLEND-3] A native control API reduces UI implementation work; it does not make arbitrary Python cheap. Patch should retain the last validated component tree and request data asynchronously rather than execute remote `draw` callbacks during a MyGo build. Host-side list templates and viewport data requests preserve the visible-item advantage across IPC.

Blender also documents that its Python integration is not thread-safe and recommends a separate process for independent Python execution. [BLEND-5] Patch's isolated helper is therefore a deliberate difference, not a claim that Blender already implements Patch's async model. Blender extension manifests separate schema version, package version, supported application versions, platforms, wheels and explained permissions. [BLEND-6] Adopt that separation and understandable permission reasons; the manifest page alone does not prove OS-level permission enforcement.

### Raycast and remote-dom: the authoring/renderer split

Raycast says its custom React reconciler produces a render tree understood by the native AppKit host. The extension author gets high-level lists, grids, details, forms and actions rather than a DOM or every AppKit type. [RAY-1, RAY-3] This is a stronger precedent for Patch's TypeScript experience than selecting a JS engine in isolation: common components, good types and coherent keyboard behavior remove more author work than exposing a low-level foreign-function interface.

Its documented Node host uses worker isolates, an event loop and a defined RPC API, while explicitly leaving normal Node file/network APIs available. [RAY-2] Patch must distinguish **renderer isolation** from **OS permission isolation**. A Node-based candidate should be assessed as a trusted-code model unless a separate enforcement design is actually implemented.

remote-dom contributes the narrower technical pattern: stable node IDs, insert/remove/property/text changes, method/event bridging and host receivers. Its optional batching and retain/release hooks make ordering and callback lifetime explicit. [REMOTE-2] Patch should borrow these ideas, not adopt a DOM polyfill or React reconciler before one is necessary. A small typed tree builder can emit the same protocol; a future JSX adapter can be layered above it.

### Neovim and Emacs: programmability without confusing immediacy with speed

Neovim's API uses handles and exposes API level, compatibility level, function `since` and deprecation metadata. Its RPC boundary cannot carry arbitrary Lua functions; API paths supporting local Lua callbacks are not automatically remotely callable. Most editor API calls are deferred to the main input loop; fast events are deliberately constrained. [NVIM-2] These are concrete reasons for Patch's opaque callback IDs and asynchronous requests.

Emacs can replace a function definition while the editor is running; `defvar` and interactive evaluation have subtleties about preserving or resetting values. [EMACS-1] The desirable DX is short feedback and inspectability, not inheriting unbounded global mutation. Patch should restore explicitly versioned extension state rather than promise that arbitrary closures, threads or native resources survive reload.

### Sublime and IntelliJ: host life is a product API

Sublime's docs make a distinction between importing a module and `plugin_loaded`, when the API is ready. `plugin_unloaded` is a cleanup hook. Asynchronous callbacks are separate from synchronous callbacks. The versioned docs currently contain both historical Python 3.3/3.8 paths and a build-4205 Python 3.14 transition, so “Sublime uses Python X” without a build is unreliable. [SUB-1, SUB-2]

IntelliJ says dynamic loading/unloading happens on the EDT under a write action, and requires unloadable extension points and resource cleanup. Services are disposed on project close/plugin unload, while declaratively registered extensions are not automatically disposed just by registration. [IDEA-2, IDEA-3] Patch can simplify this by requiring every runtime registration to belong to an activation or pane scope that the host revokes wholesale.

### Kero, GoRex and Godiff: source-grounded design references

Kero documents project-grouped sessions in a sidebar, status that distinguishes working/waiting/idle/error/exited, and a command to jump to a session needing attention. GoRex documents workspace-like tabs, split panes as cards, per-pane program/directory headers, drag resizing, zoom, close, tab reordering, and keyboard navigation/divider movement. Both describe session servers distinct from the window. [REF-1, REF-2]

The current shell source extends **Kero-style Sidebar / GoRex-style title-bar Tabs** into Above/Left/Right choices, still showing one workspace hierarchy at a time. Mode, side and collapse state are independent; navigation does not change automatically with viewport width. Compact same-baseline headers, unboxed actions and invisible gutters remain the design contract. Earlier native acceptance and latest refinement status are distinguished in the coordinator's [report](verification.md). Pane identity remains separate from placement/process identity; layout persistence does not imply a terminal daemon. Reference findings here remain source/doc readings, while visual acceptance comes from the native judge.

Kero's pinned `panes.go` implements header-origin pane dragging, edge targets and a center **swap** target (`zoneAt`/`movePane`). Patch's corrected M0 contract follows that interaction: center swaps (`SwapPanels` in the coordinated core), edges reposition/split. The earlier tab-merge proposal was an unrequested reinterpretation. Existing multi-panel groups are preserved through a compact pane-header selector; new panels open as visible panes. A broader grouped-panel UX remains a product decision. Kero was inspected at `fd335a5551ce4e2d2d14e1e7d257b37214097cf4` (2026-11-03); GoRex at `b206e67bf0ac422afebc5c9e701a38dd67f2c837` (2026-10-08). The inspected GoRex root tree contains no root license file; it is a behavior/visual reference, not source to copy under an assumed license. [REF-1, REF-2]

The shell prioritizes real terminal/files/Git work. Earlier [native evidence](verification.md#verified) covers restart, folder selection and friendly non-Git state; the [refinement report](verification.md#refinement-batch-tab-style-diff-review-glyph-size) establishes the latest font/diff PASS and popup-action limits. **Godiff** supplies visual/native-diff learning through coordinator-supplied revision [`da918821`](https://github.com/egoist/godiff/commit/da918821) and its [README](https://github.com/egoist/godiff#readme). Patch acceptance comes from its own native evidence, not that reference. [REF-3]

## 5. Runtime options

| Option | Useful documented properties | Integration / DX cost | Recommendation |
|---|---|---|---|
| **goja in a Go helper** | Pure Go ECMAScript 5.1(+) implementation; multiple VMs; interruption; host-supplied concurrency. | A runtime is single-goroutine-at-a-time; no general Node compatibility or built-in browser APIs/event loop. Transpilation handles syntax, not missing runtime semantics. No documented hard per-VM heap ceiling comparable to QuickJS was established here. [JS-1] | **First candidate**, because it preserves Go-only builds and a narrow host API. Validate a precise JS target, promises, module bundle and source maps. |
| **QuickJS helper** | Small C interpreter; inspected manual version 2026-06-04 documents near-complete ES2025, modules, memory/stack limits and interrupt callback. | C build/distribution or another embedding layer; host owns pending jobs, async I/O and modules. ECMA402 is absent in that manual. Bytecode is engine-version-bound and not validated for untrusted loading. [JS-2, JS-3] | Serious alternative when modern language support or hard VM allocation limits justify toolchain cost. Ship JS source bundles, not public bytecode packages. |
| **Node helper** | Familiar TS/JS package ecosystem, event loop, worker-based execution; official V8 Inspector supports debugger/profiler attachment. | Managed runtime distribution and lifecycle; larger footprint is plausible but **unmeasured here**. Node built-ins mean a thin RPC API alone cannot enforce file/network restrictions. [RAY-2, NODE-1] | Comparison baseline and likely choice if real extension requirements depend on Node packages or full debugging. Do not emulate Node piecemeal to save a hypothetical memory budget. |
| **GopherLua** | Lua 5.1-oriented VM/compiler in Go; Go binding API, selectable standard libraries, context cancellation. | `LState` is not goroutine-safe; cancellation has documented overhead. Different package/tooling audience; Lua 5.1 is not Lua 5.4 or LuaJIT compatibility. [LUA-1, NVIM-1] | Good alternative for personal scripting if users prefer Lua; avoid supporting two public SDKs before demand exists. |
| **WASM via wazero** | Go runtime; imports define host access; configurable linear-memory page limit and close-on-context cancellation. [WASM-1, WASM-2] | Compilation/toolchain loop, serialized host ABI, debugging and async adapters. Linear memory is not total process memory. Imports can still block or expose authority. Zed's component/WASIp2 target is not proof of compatibility with a chosen wazero ABI. | Later compute/provider surface. Benchmark an explicit core-WASM/import ABI before choosing component-model tooling. |
| **Compiled Go sidecar** | Ordinary executable; `os/exec`/pipes and versioned messages decouple host/toolchain; restartable. | Build/distribute each target, startup/IPC overhead and explicit state restoration. Runs with user's OS authority unless separately sandboxed. [GO-1, GO-3] | Useful for CPU work, native integrations or existing tools; not the default authoring loop. |
| **Go `plugin` package** | Direct symbols and shared data. | Cannot close/unload; platform limits; same toolchain, build flags and dependency sources required. Official docs explicitly suggest IPC for many uses. [GO-1] | Reject for the public reloadable extension surface. |
| **QML as extension language** | A mature declarative UI language with JS bindings. | Requires Qt integration or reimplementing its semantics; duplicates MyGo's UI responsibilities. [QT-1, QT-2] | Learn from its ownership model; not a runtime recommendation for this Go/MyGo brief. |

**Interpreted is a DX preference, not an automatic performance result.** TypeScript still needs a compile/bundle step. QuickJS compiles source to bytecode internally; WASM is compiled input; Go sidecars can be rebuilt/restarted. The user-visible goal is a short, reliable edit-to-feedback loop. Measure that loop and the host's response while extensions misbehave, not the language label.

### Proposed goja host shape

Use a dedicated `patch-extension-host` Go executable with no MyGo imports, one active extension per process initially, one VM event loop per process. This provides a simple restart and accounting boundary and keeps UI toolkit initialization out of helper startup. Every incoming completion is enqueued on that VM's owner loop; host services return promises and never call JS from arbitrary Go goroutines. `Interrupt` can stop executing JS, but a blocking Go function exposed into the VM still needs its own cancellation. [JS-1]

Expose only the Patch SDK, a small documented JS standard-library target and bundled pure-JS dependencies. Do not export reflection-backed arbitrary Go objects, a filesystem `require`, native modules, raw OS calls or broad Node shims. Use the same protocol for a Node/QuickJS prototype so runtime comparison does not become a UI rewrite.

A process boundary is **crash containment**, not a proven OS sandbox or a hard memory quota. Go memory limits and process monitoring must not be advertised as a hard per-VM cap. Pure goja host imports can be capability-checked, but a trusted native/Node sidecar with OS APIs has broader authority. Keep that distinction visible in manifests and installation UX.

## 6. Decisions to validate, not assume

1. **JS compatibility:** exercise the proposed SDK, async cancellation, timers, source maps, one real pure-JS parser and one network-backed extension in each candidate. Record unsupported APIs; do not promise arbitrary npm compatibility.
2. **UI adapter:** preserve text composition, focus and list anchors across keyed updates and pane reparenting. MyGo's parent-derived identity makes docking a specific experiment, not a solved detail.
3. **Resources:** compare cold activation, 10 installed/0 active and 10 active lightweight extensions, repeated reload, CPU-heavy work and a stalled producer. Report whole process tree and GPU footprint separately from Go heap.
4. **Hot reload:** test syntax failure, activation failure, snapshot version mismatch, cancellation and late replies. Keep old rendered content and user-owned host state recoverable without claiming rollback of external effects.
5. **Virtualization:** a large dataset must not become a large serialized component tree. Measure request windows, overscan, resident rows, bytes transferred, first result and scrolling while a provider is slow.
6. **Webview:** separately exercise lazy creation, page failure, hidden retention/destruction, keyboard return and overlay/capture behavior on the actual pinned platform implementation.

These are bounded runtime-spike questions for a later milestone. They do not authorize adding the ecosystem runtime to the current initial-shell build.

## 7. Primary source register

Directly researched links were accessed **2026-11-03**; coordinator-supplied HIG and Godiff references are explicitly identified below. Bracketed source IDs elsewhere in these documents refer to this register. Dates in the evidence column are upstream publication/measurement dates when explicitly documented; otherwise the source is rolling.

| ID | Official source | Evidence / revision |
|---|---|---|
| MYGO-1 | [MyGo README](https://github.com/egoist/mygo/blob/6842ad5e0aa2cd168166e0a4ae12c0cd0f527a3d/README.md), [requirements](https://github.com/egoist/mygo/blob/6842ad5e0aa2cd168166e0a4ae12c0cd0f527a3d/docs/README.md) | Native/web surface distinction, platform dependencies, Go requirement; pinned 2026-11-03 commit. |
| MYGO-2 | [Views](https://github.com/egoist/mygo/blob/6842ad5e0aa2cd168166e0a4ae12c0cd0f527a3d/docs/ui/views.md), [migration/identity](https://github.com/egoist/mygo/blob/6842ad5e0aa2cd168166e0a4ae12c0cd0f527a3d/docs/ui/migration.md) | Build lifetime, keys, handles, updates. |
| MYGO-3 | [Split view](https://github.com/egoist/mygo/blob/6842ad5e0aa2cd168166e0a4ae12c0cd0f527a3d/docs/ui/split.md) | Pointer and keyboard resizing, DIP and percentage size semantics. |
| MYGO-4 | [Tabs](https://github.com/egoist/mygo/blob/6842ad5e0aa2cd168166e0a4ae12c0cd0f527a3d/docs/ui/tabs.md), [UI catalogue](https://github.com/egoist/mygo/blob/6842ad5e0aa2cd168166e0a4ae12c0cd0f527a3d/docs/ui/README.md) | Navigation/control availability; not a docking manager. |
| MYGO-5 | [List](https://github.com/egoist/mygo/blob/6842ad5e0aa2cd168166e0a4ae12c0cd0f527a3d/docs/ui/list.md) | Visible rows, variable heights, item keys, focus and scroll anchoring. |
| MYGO-6 | [Web view](https://github.com/egoist/mygo/blob/6842ad5e0aa2cd168166e0a4ae12c0cd0f527a3d/docs/ui/web-view.md) | Page lifetime, trust, focus, clipping and capture limits. |
| MYGO-7 | [Architecture](https://github.com/egoist/mygo/blob/6842ad5e0aa2cd168166e0a4ae12c0cd0f527a3d/docs/architecture.md) | Main thread, native backends, event-driven frames and source-reported measurements. |
| MYGO-8 | [Terminal plugin guide](https://github.com/egoist/mygo/blob/6842ad5e0aa2cd168166e0a4ae12c0cd0f527a3d/docs/plugins/terminal.md) | Native terminal integration; separate from extension host. |
| MYGO-9 | [Native UI performance](https://github.com/egoist/mygo/blob/6842ad5e0aa2cd168166e0a4ae12c0cd0f527a3d/docs/ui/performance.md) | Upstream headless/CPU fixture measurements, 2026-10-08 and 2026-10-14; not reproduced. |
| REF-1 | [Kero README](https://github.com/egoist/kero/blob/fd335a5551ce4e2d2d14e1e7d257b37214097cf4/README.md), [pane docking](https://github.com/egoist/kero/blob/fd335a5551ce4e2d2d14e1e7d257b37214097cf4/panes.go) | Project sidebar, statuses, independent sessions, center-swap/edge-dock behavior; pinned 2026-11-03 revision. |
| REF-2 | [GoRex README](https://github.com/egoist/gorex/blob/b206e67bf0ac422afebc5c9e701a38dd67f2c837/README.md), [root tree](https://github.com/egoist/gorex/tree/b206e67bf0ac422afebc5c9e701a38dd67f2c837) | Tabs/pane cards, resizing, keyboard commands, session server and limits; pinned 2026-10-08 revision. |
| REF-3 | [Godiff README](https://github.com/egoist/godiff#readme), [source revision da918821](https://github.com/egoist/godiff/commit/da918821) | Coordinator/diff-worker-supplied native-diff visual/implementation reference. No new independent source survey or Patch native-pass claim. |
| VSC-1 | [Extension host](https://code.visualstudio.com/api/advanced-topics/extension-host) | Node/browser/remote hosts and `extensionKind`. |
| VSC-2 | [Contribution points](https://code.visualstudio.com/api/references/contribution-points) | Static manifest declarations. |
| VSC-3 | [Activation events](https://code.visualstudio.com/api/references/activation-events) | Lazy activation; 1.74 inferred activation notes. |
| VSC-4 | [Extension capabilities](https://code.visualstudio.com/api/extension-capabilities/overview) | No access to workbench DOM; supported extension surface. |
| VSC-5 | [Webview guide](https://code.visualstudio.com/api/extension-guides/webview) | Resource cost, messages, state, context retention, CSP. |
| VSC-6 | [Settings](https://code.visualstudio.com/docs/configure/settings), [keybindings](https://code.visualstudio.com/docs/configure/keybindings) | User/workspace precedence and inspectable binding rules; settings source updated 2026-10-28. |
| NVIM-1 | [Lua documentation](https://neovim.io/doc/user/lua.html), [source](https://github.com/neovim/neovim/blob/master/runtime/doc/lua.txt) | Lua 5.1 contract, LuaJIT, event-loop and thread restrictions. |
| NVIM-2 | [API documentation](https://neovim.io/doc/user/api.html), [source](https://github.com/neovim/neovim/blob/master/runtime/doc/api.txt) | RPC, handles, fast/deferred calls, API evolution. |
| ZED-1 | [Developing extensions](https://zed.dev/docs/extensions/developing-extensions) | Current contributions, Rust/WASM, WASIp2 target and dev loading. |
| ZED-2 | [Extension capabilities](https://zed.dev/docs/extensions/capabilities) | Execution/download/npm permission methods and user restrictions. |
| EMACS-1 | [Evaluating Lisp](https://www.gnu.org/software/emacs/manual/html_node/emacs/Lisp-Eval.html) | Live definition evaluation and state subtleties. |
| EMACS-2 | [Threads](https://www.gnu.org/software/emacs/manual/html_node/elisp/Threads.html) | Shared memory, mostly cooperative scheduling. |
| EMACS-3 | [Asynchronous processes](https://www.gnu.org/software/emacs/manual/html_node/elisp/Asynchronous-Processes.html) | Process/filter model and partial asynchrony. |
| SUB-1 | [API environments](https://www.sublimetext.com/docs/api_environments.html) | Embedded Python host and build-specific version transitions. |
| SUB-2 | [API reference](https://www.sublimetext.com/docs/api_reference.html) | Lifecycle, thread-safe APIs, async callbacks. |
| IDEA-1 | [Threading model](https://plugins.jetbrains.com/docs/intellij/threading-model.html) | EDT/background tasks and shared model access. |
| IDEA-2 | [Dynamic plugins](https://plugins.jetbrains.com/docs/intellij/dynamic-plugins.html) | Unloading restrictions and resource cleanup; page dated 2024-10-31. |
| IDEA-3 | [Disposer and Disposable](https://plugins.jetbrains.com/docs/intellij/disposers.html) | Service ownership and extension disposal distinctions. |
| RAY-1 | [FAQ](https://developers.raycast.com/misc/faq) | Custom React reconciler to AppKit, no HTML/CSS requirement. |
| RAY-2 | [Security/runtime model](https://developers.raycast.com/information/security) | Child Node process, isolates/RPC; explicitly not further file/network sandboxed. |
| RAY-3 | [User interface](https://developers.raycast.com/api-reference/user-interface), [lifecycle](https://developers.raycast.com/information/lifecycle) | Supported native component catalogue and command lifecycle. |
| BLEND-1 | [Blender Panel API](https://docs.blender.org/api/current/bpy.types.Panel.html) | Native panel registration, location, context and draw contract. |
| BLEND-2 | [Blender UILayout API](https://docs.blender.org/api/current/bpy.types.UILayout.html) | Host layout, property and operator controls. |
| BLEND-3 | [Blender UIList API](https://docs.blender.org/api/current/bpy.types.UIList.html) | Visible-item draw callback; explicit warning against expensive draw traversal. |
| BLEND-4 | [Blender areas](https://docs.blender.org/manual/en/latest/interface/window_system/areas.html) | Area docking, split/join, resize, cancel, swap and maximize. |
| BLEND-5 | [Blender Python threading limitations](https://docs.blender.org/api/current/info_gotchas_threading.html) | Python integration not thread-safe; independent-process alternative. |
| BLEND-6 | [Blender extension manifest](https://docs.blender.org/manual/en/latest/advanced/extensions/getting_started.html) | Schema/package/app versions, platform packaging and permission reasons. |
| REMOTE-1 | [remote-dom README](https://github.com/Shopify/remote-dom/blob/main/README.md) | Remote environment/receiver design. |
| REMOTE-2 | [Core API](https://github.com/Shopify/remote-dom/blob/main/packages/core/README.md), [wire types](https://github.com/Shopify/remote-dom/blob/main/packages/core/source/types.ts) | Mutations, method/events, batching and retain/release. |
| QT-1 | [Qt Quick performance](https://doc.qt.io/qt-6/qtquick-performance.html) | Frame work budget, asynchronous design, profiling and binding costs. |
| QT-2 | [QML architecture](https://doc.qt.io/qt-6/qtqml-index.html) | Declarative object system and JavaScript integration. |
| JS-1 | [goja README](https://github.com/dop251/goja/blob/master/README.md), [runtime API](https://pkg.go.dev/github.com/dop251/goja) | Pure Go, ECMAScript scope, goroutine restrictions, interruption and host-provided event loop. |
| JS-2 | [QuickJS manual](https://bellard.org/quickjs/quickjs.html) | Manual version 2026-06-04; ES2025 support, C API, limits, bytecode restrictions. |
| JS-3 | [QuickJS public header](https://github.com/bellard/quickjs/blob/master/quickjs.h) | Runtime/memory/stack/interrupt/module and pending-job APIs. |
| NODE-1 | [Node debugging](https://nodejs.org/api/debugger.html), [Inspector](https://nodejs.org/api/inspector.html) | V8 Inspector debugger/profiler integration; Context7 official-source lookup. |
| LUA-1 | [GopherLua README](https://github.com/yuin/gopher-lua/blob/master/README.md) | Go embedding, restricted libraries, `LState`, cancellation and measured-example caveat. |
| WASM-1 | [wazero documentation](https://wazero.io/docs/) | Go runtime, WASM execution, host imports. |
| WASM-2 | [wazero runtime config](https://github.com/tetratelabs/wazero/blob/main/config.go) | `WithMemoryLimitPages`, `WithCloseOnContextDone`, explicit environment. |
| GO-1 | [Go plugin source](https://github.com/golang/go/blob/master/src/plugin/plugin.go) | No unload, portability/build compatibility, IPC alternative. |
| GO-2 | [Go toolchains](https://go.dev/doc/toolchain) | Automatic toolchain selection and module/toolchain requirements. |
| GO-3 | [Go exec source](https://github.com/golang/go/blob/master/src/os/exec/exec.go) | Executable/pipe process model; no implicit shell. |
| HIG-1 | [Apple split views](https://developer.apple.com/design/human-interface-guidelines/split-views), [layout](https://developer.apple.com/design/human-interface-guidelines/layout) | Coordinator-supplied research: nested splits, usable minima/dividers and non-destructive adaptive resizing; provided 2026-11-03. |
