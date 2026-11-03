# Patch — extensible native development environment

**Version:** 0.1 research proposal · **Date:** 2026-11-03 · **Status:** M0 and the latest native refinements have scoped macOS Apple-silicon acceptance. Extension architecture and performance budgets remain proposals.

## 1. Product decision and delivery boundary

Patch is a **native, configurable workspace for development tools**: workspace navigation and resizable/repositionable panes containing terminals, files, review tools, task output and extension-defined interfaces. The environment should feel coherent even when its contents come from different authors.

Use **Go + [MyGo](https://github.com/egoist/mygo)** for the shell and native components. Here, native means **MyGo's custom GPU-drawn UI integrated with platform windows, text/input and accessibility services**, not a web frontend and not necessarily OS widget controls. System webviews are an explicit, on-demand extension surface. [MYGO-1, MYGO-7]

**Recommended extension direction:** TypeScript authoring compiled to JavaScript; a typed, declarative native component API; asynchronous host services; a restartable extension-host process. Evaluate **goja in a small Go helper first**, against Node and QuickJS, before making the runtime a public compatibility promise. The language does not own the hot path: the host owns layout, focus, pointer handling, scroll, text composition and rendering. Native components alone do not make arbitrary extension code fast. [JS-1–3, NODE-1, BLEND-3, QT-1]

### What this work delivers now

The authorized build is **research plus an initial shell**. Initial-shell coordination targets:

- Native Go/MyGo window with **Above, Left or Right** navigation placement, selected through **View → Customize View → Tab Style**. Above uses title-bar tabs; Left/Right use a sidebar.
- A nested split tree with stable pane identities, resizing, edge docking, center swaps and persisted configuration/layout. Existing multi-panel groups remain readable through a compact header selector; opening a new panel creates a visible pane.
- Real local terminals, files and Git/project surfaces as the primary tools. A singleton native-control gallery is a developer reference, not a main-navigation product surface.
- An on-demand system-webview pane demonstrating the optional surface.

The coordinator accepted M0 and the subsequent navigation, terminal-font and native-diff refinements within the scope recorded in [Native verification](verification.md#refinement-batch-tab-style-diff-review-glyph-size) and [§14](#14-acceptance-criteria). At research start the supplied baseline was README-only `main` at `a2f3d1c`. The root README and implementation documentation define the runnable interface. This document incorporates coordinator-supplied evidence; its author did not independently run the application.

**Design revision status:** the corrected M0 baseline received **Opus 5.5 xhigh visual PASS** against Kero/GoRex; the latest native refinement acceptance is also **PASS**. The coordinator inspected the current diff, Above and Right Side images. [Current captures](verification.md#current-screenshots) and the [refinement results](verification.md#refinement-batch-tab-style-diff-review-glyph-size) cover interaction states, alignment, corners and diff readability; popup-action and OS-composited pixel limits remain explicit.

The initial shell does **not** establish a public extension SDK, interpreted execution, permissions enforcement, extension packaging, hot reload, remote workspace execution or persistent sessions across application exit. Every `@patch/*` import, extension manifest, protocol and CLI example below is a **proposed future API**, not an available command or current configuration format. Current internal Go APIs and persisted JSON may differ while the model is being proven.

### Evidence convention

Bracketed IDs link conceptually to the dated [primary source register](research.md#7-primary-source-register). Research distinguishes source evidence, upstream measurements, recommendations and unknowns. **All performance budgets in §13 are proposed targets awaiting measurement.** Separately labeled M0 diagnostics are observations, not budget passes. No competitive performance claim is made.

## 2. Problem, outcomes and principles

Developers assemble environments from terminals, editors, source-control tools, agents and project dashboards. Fixed sidebars and special-case panels make that environment difficult to adapt. Fully custom extension HTML can restore flexibility but transfers design, focus, accessibility, rendering and memory management to each extension. A native host can offer richer panel composition without requiring every author to build another application.

### Desired outcomes

1. Arrange tools around the task, then return to the same useful arrangement.
2. Keep typing, scrolling and resizing responsive while tools stream data or extensions compute.
3. Build a useful custom panel with a small amount of typed code and immediate feedback.
4. Make settings, commands, keybindings and appearance inspectable and shareable.
5. Recover a failed tool or extension without losing the surrounding workspace or unrelated work.

### Principles

- **Host interaction, extension intent.** Extensions name actions, data and native controls; the host supplies platform behavior and rendering.
- **Configuration is a product surface.** Discoverable UI and declarative files express the same supported settings. A keyboard user can arrange the workspace too.
- **Stable identity before decoration.** Moving a pane changes its location, not its terminal session, query, selection or draft.
- **One coherent design language.** Shared density, typography, tokens and interaction states; customization remains broad without requiring per-extension CSS.
- **Lazy work, bounded work.** Installed is not activated; hidden is not continuously rendered; producers cannot enqueue unlimited UI work.
- **Simple public contracts.** Start with commands, services, contributions and a small native component vocabulary. Add a general-purpose adapter only when a real extension needs it.

## 3. Users and workflows

| Persona | Job | Representative workflow | Success condition |
|---|---|---|---|
| **Project developer** | Work across source, terminal and feedback. | Open a project; keep files on the left, editor/review centrally, tests below; resize for a laptop and restore later. | No lost pane state; core actions are accessible from keyboard and pointer. |
| **Multi-session / agent user** | Monitor independent work without constant switching. | Choose a Kero-like project/status sidebar or GoRex-like title-bar tabs alongside terminal/task panes; jump to the tool needing attention. | One clear workspace-navigation hierarchy; readable status and preserved context. |
| **Workflow customizer** | Make the environment match personal/project conventions. | Edit theme tokens, bindings and a layout preset; preview changes; inspect an overridden setting; restore defaults. | Changes are reversible, validated and explained by source/precedence. |
| **Extension author** | Add a useful tool quickly. | Scaffold a TS panel; expose a list/form and command; save; inspect a native preview; fix an error without restarting Patch. | Short measured feedback loop, useful stack traces and explicit resource/state ownership. |
| **Tool integrator** | Reuse a language server, CLI or expensive analyzer. | Register a service provider; run work in a process; stream paginated results into native components. | Cancellation and bounded data flow; protocol independent of implementation language. |

### Core journeys

**J1 — Arrange and return.** Restore the last workspace or choose a project folder, add two terminals and a files/project pane, split horizontally and vertically, move a pane to an edge, swap two panes with a center drop, resize, close/reopen Patch. New panels appear as visible panes. Layout and supported pane descriptors restore. A shell descriptor may start a new shell after app exit; preserving a running process requires a later session service.

**J2 — Focus without a mouse.** Switch workspaces through the selected navigation mode, focus a neighboring pane, resize a split, open a pane's actions, zoom/restore and use the command palette. Text controls and terminals retain their expected editing keys. Escape cancels a docking operation or dismisses the active overlay predictably.

**J3 — Author a native tool.** Register a command and panel, supply asynchronous list data, update a field, save source and reload. The panel stays in place, focus and compatible native state survive, and old-generation replies cannot overwrite new content.

**J4 — Work through failure.** A provider is slow, a request is canceled, then an extension crashes. The host continues operating; the affected pane offers retry/reload and access to diagnostics. A compile error leaves the last working version available.

**J5 — Use an optional web tool.** Explicitly open a webview panel; its page loads inside the pane, native chrome remains in control, and keyboard focus can return to the workspace. Closing releases the page. Other panes work when the page fails or a webview runtime is unavailable.

## 4. Scope and milestones

Milestones sequence implementation; they do not reduce the product to a shell-only MVP. Each later milestone requires separate execution after the current research/shell task.

| Milestone | Scope | Exit evidence |
|---|---|---|
| **M0 — Research and initial shell (accepted, scoped)** | Above/Left/Right navigation, stable panes, split/resize/edge-dock/center-swap behavior, persistence, real terminal/files/native-diff surfaces, opt-in webview and developer-only singleton gallery. | Baseline and latest macOS native refinements passed within the [recorded scope](verification.md#refinement-batch-tab-style-diff-review-glyph-size). Popup diff actions retain source/headless coverage only. Windows x64 packaging is built/inspected; Windows runtime remains unverified. |
| **M1 — Extension architecture spike (proposed)** | One command-only extension and one range-backed native list/form, Go helper/goja candidate, same protocol tested with Node/QuickJS; cancellation, bad extension, reload and identity probes. | Measured runtime/DX comparison, host responsiveness evidence and an explicit runtime decision. No marketplace or broad compatibility claim. |
| **M2 — Usable local extension SDK (proposed)** | Typed SDK, contribution registration, service scopes, native protocol, dev CLI, logging/source maps, local install, explained capabilities, stateful reload and recovery. | Independent author builds the reference tool from docs; J3/J4 pass; narrow API/version contract is published. |
| **M3 — Complete configurable environment (proposed)** | Settings editor/explain, theme/keybinding/layout-preset contributions, provider integrations, accessibility and scale work, well-defined webview extension bridge. | Main personas complete J1–J5; limits and supported platforms measured; settings and UI remain consistent across extensions. |
| **M4 — Distribution and advanced providers (proposed)** | Versioned packages, update/rollback, compatibility checks, curated discovery if warranted; compiled/WASM provider support only where demonstrated useful. | Install/update/rollback and dependency failures preserve work; documentation and support boundaries are clear. |

### Outside the current build

A full IDE feature set, language-server/debugger compatibility, remote execution, collaboration, a terminal daemon, marketplace business operations and VS Code binary/API compatibility are not part of M0. They are separate product decisions, not requirements hidden inside “extensible.” A native editor may be a first-party pane without committing Patch to replacing every editor immediately.

## 5. Workspace and native shell requirements

### 5.1 Layout model

**M0 scope:** the [§14 A0 table](#14-acceptance-criteria) is the authoritative current-shell acceptance list. L-1–L-7 describe the product model; their future portions are marked below and do not add undocumented M0 requirements.

**Terminology:** a *pane* is `workspace.Panel` in the M0 implementation; its *panel type* is `Kind`.

**L-1.** Preserve stable workspace, group and pane IDs across layout changes. A split has an orientation, ratio and two children; a group leaf contains panel descriptors and an active panel. Runtime instances, such as a running terminal, have lifetimes independent of placement. In M0, Sidebar and title-bar Tabs are alternative presentations of `State.Workspaces`, not simultaneous duplicate navigation or separate identity layers. Persisted group nodes embed panel descriptors. Existing multi-panel groups remain supported through a compact selector within the pane header, without another tab row. Opening a new panel creates a visible pane; a broader grouped-panel product experience remains an open product decision rather than an M0 requirement.

**L-2.** Edge docking repositions a pane by splitting at the destination edge. **Center docking swaps the source and destination panes**, following Kero; the coordinated core operation is `SwapPanels`. It must be possible to cancel before commit. Drag preview, target highlight and final position agree, with keyboard access to the supported transformations. Center-drop tab merging was an unrequested reinterpretation and is not the M0 contract. [REF-1]

**L-3.** Moving a pane preserves its ID and native model. Empty groups are removed and redundant splits collapse. Invalid cycles, duplicate pane placement and invalid active-pane references are rejected before persistence. In M0, a direct user drag or keyboard move focuses the moved pane at its destination. **M2+ requirement:** a programmatic/background extension move of an unfocused pane must preserve the user's current focus rather than steal it.

**L-4.** Users can resize with the pointer and keyboard, zoom/restore, close and reposition panes through explicit actions, with M0 coverage defined by §14. M0's double-click/Enter split reset to 0.5 is [native-verified](verification.md#verified). **M3:** complete the customization command surface, including rename and broader equalize actions not documented by M0; the current core has no rename API. Minimum sizes keep headers and key content usable. M0 uses **invisible gutter grab regions with no divider hairlines at rest**; the cursor and active interaction communicate resizing. Preserve the user's preferred ratios when narrow sizes require clamping, recovering them when space returns. Platform split-view guidance informs usable sizes/input, not a requirement to add visible lines contrary to the user's references. [MYGO-3, REF-2, HIG-1]

**L-5.** Pane closing is distinct from tab switching/hiding. M0 documents its local terminal close behavior; hidden panes keep their processes, and layout restoration uses descriptors and stable IDs rather than serialized UI objects. **M3+:** configurable close policies and multiwindow/floating-window docking extend this model; they are not M0 obligations.

**L-6.** The native **View → Customize View → Tab Style** menu selects **Above**, **Left** or **Right**. Above uses title-bar Tabs; Left/Right use Sidebar mode. Mode, side (`sidebarPosition`) and visibility persist separately. The sidebar toggle only collapses/expands Sidebar mode, preserving its side; it is absent in Above mode. Viewport resizing never automatically changes navigation mode. Keep one presentation of the workspace hierarchy, with no duplicate navigation or stacked tab rows. The hostname block is removed; the hover-close X remains aligned within the workspace pill. Modes, independent visibility, alignment and whole-pill focus/pressed states are [native-verified](verification.md#verified-natively).

Sidebar side and visibility are current M0 source capabilities, not M3 proposals. The remaining M3 customization surface includes pane-header density, default pane placement, component typography/spacing, per-pane options and named workspace presets. All content pane types participate in the same move/split/swap model; a first-party terminal or project pane is not permanently tied to a special region. Grouped-panel navigation beyond preserving existing groups needs an explicit product decision. M0's implementation documentation defines its actual configuration.

**L-7.** Saved layouts are versioned and written atomically. M0 validates loading/saving and reports invalid configuration/layout without destroying the last valid data, as specified by A0-3. **M3:** live external edits retain the last valid live layout on error rather than silently overwriting the user's file. **M2+:** missing extension panes restore as placeholders with reinstall/remove actions rather than disappearing.

### 5.2 Design and usability

Use the supplied references directly: **Kero for Sidebar navigation and center-swap/edge-docking interaction; GoRex for title-bar Tabs and pane-header composition**. Match the chosen reference mode instead of combining its navigation with the other mode into extra UI layers. Both use a native host and describe pane/session state separately from layout. Blender remains an architectural precedent for independently configurable areas and script-authored host controls, not authorization to reinterpret the requested shell. [REF-1, REF-2, BLEND-1–4]

The visual direction is restrained and information-dense: clear primary work area, quieter navigation, semantic active/focus states, balanced padding and related corner radii. Each pane has **one compact header**, with its title and muted path on the same baseline and unboxed action icons. Existing group selection fits that header rather than creating another tab row. Pane gutters remain visually clear at rest. Theme tokens control background levels, text, accent, status and focus without imposing extra borders or decoration.

- Prioritize real terminals, files and Git/project work. The native-control gallery is a developer-only singleton reference; it does not occupy main navigation, become the default work surface or open duplicate galleries.
- A fresh Finder launch restores a valid last workspace or offers folder selection. Do not infer the filesystem root as the project from a process working directory of `/`.
- A non-repository Changes pane explains that the folder is not a Git repository and provides a useful next action; raw Git command errors are diagnostic detail, not the normal empty state.
- Terminal fallback now uses the **non-Mono JetBrains Regular font as a secondary face**, after Menlo on macOS and the system primary on Windows. This replaces the earlier Symbols Mono sizing approach. The coordinator reports native PASS for the actual spaced prompt's icon size; the user explicitly chose Kero-like sizing and accepted ink overhang when an icon immediately adjoins non-space text. This is a deliberate sizing trade-off, not a claim of universal glyph containment or Windows font verification. Current evidence belongs in [verification.md](verification.md).
- Files uses root-cache GPU compositing to address MyGo's nested-clip limitation; Files and Changes corners are native-verified. Changes uses native `editor.DiffView` file cards with colored lines, word highlights and staged/unstaged/untracked context, informed by [Godiff](https://github.com/egoist/godiff). The highlighter crash and subsequent row-height/EOF/color issues were fixed and [confirmed natively](verification.md#refinement-batch-tab-style-diff-review-glyph-size). Popup Unified/Split/Collapse all/Expand all actions remain source/headless-checked, not native-tooling-verified. [REF-3]

- Use native text/input behavior, readable long labels, tooltips for truncated identity, and accessible labels for icon-only actions.
- Differentiate active workspace, active pane, keyboard focus, running status and attention. Do not encode all five as the same accent tint or depend on color alone.
- Preserve IME composition, selection and undo during ordinary updates. Tab traversal, platform editing shortcuts and context menus should feel consistent across panes.
- Show loading, empty, failed, disabled, read-only and disconnected states within the same layout, with a useful next action.
- Motion explains a state change; resizing tracks the pointer directly. Reduced-motion settings suppress nonessential transitions. Idle UI does not animate solely to indicate life.
- Native accessibility is an acceptance requirement, not assumed because MyGo advertises a bridge. Test focus order, names, tab/split semantics, text fields and one virtual collection on each platform claimed supported. [MYGO-4, MYGO-7]

## 6. Extension product model

### 6.1 Contributions

The manifest is readable without activating code. Contribution IDs are namespaced by publisher and extension. Each registration has one owning extension; conflicting IDs fail with a clear diagnostic rather than “last extension wins.”

| Contribution | Static declaration | Runtime behavior |
|---|---|---|
| **Commands** | ID, title, category, argument schema, contextual availability. | Async handler or host command alias; invokable from palette, menu, binding or API. |
| **Panels** | Type ID, title/icon, default placement hint, singleton/multi-instance, supported native/web surface. | Factory creates content for a specific pane instance; host owns actual placement. |
| **Menus/actions** | Slot, command ID, order/group, declarative `when`. | Availability derives from host context; no synchronous extension callback during menu layout. |
| **Settings** | Schema, default, description, scope, reload requirement. | Typed resolved reads and change notifications. |
| **Themes/icons** | Semantic tokens and packaged asset references. | Applied by host without running extension logic. |
| **Keymaps/layout presets** | Binding rules and versioned layout templates. | User overrides and explicit preset application. |
| **Services/providers** | Versioned service ID, request/result schemas, scope. | Async request, stream or event interface; no remote object graph. |

Default activation is inferred from invoking a contributed command, opening/restoring its visible panel or requesting its service. Explicit activation handles particular workspace events or a deliberately requested background task. Theme-only/keymap-only extensions need no process. Do not activate every installed extension at startup. This follows VS Code's contribution/lazy-activation separation. [VSC-2, VSC-3]

### 6.2 Service ownership

First host services: workspace/resource metadata and bounded reads, commands, configuration, extension-scoped storage, notifications, panel opening, diagnostics and approved process/network operations. URI/resource handles avoid forcing all resources to be local path strings.

Scopes are **application**, **workspace**, **extension activation** and **pane instance**. Subscriptions, callbacks, requests and owned jobs attach to a scope. Closing a workspace cancels its requests; closing one pane does not cancel an unrelated pane; reloading one extension revokes that activation's registrations. A host-owned terminal session can outlive the panel factory when its explicit lifecycle requires it.

Cross-extension APIs use registered versioned service contracts and explicit dependencies. A provider's own permissions do not silently become a caller's permissions; services must validate caller identity where their operation delegates authority. Do not introduce a generic synchronous service locator returning foreign mutable objects.

### 6.3 Runtime recommendation and decision gate

M1 should implement a dedicated Go helper with one goja VM/active extension per process initially. The helper must not import MyGo. It owns an event loop; all VM execution and promise completion happen there. CPU work and I/O remain outside the UI thread. Installed-but-inactive extensions start no helper.

Reasons to start here are the Go-only build path and a narrow embeddable API, **not measured speed or footprint**. The JS target, transpilation output, module bundle format and standard APIs must be named precisely. goja describes ECMAScript 5.1(+) rather than full Node compatibility and leaves event-loop APIs to the host. [JS-1]

**Choose Node instead** if representative integrations need Node built-ins/npm packages or debugger/profiler tooling that materially improves authoring, and the measured resource cost is acceptable. Node's Inspector is a real DX advantage; reproducing that stack is not free. **Choose QuickJS instead** if its modern language support and VM allocation limits justify C/embedding and debugger integration work. **Choose Lua** only if the intended authors prefer it enough to warrant that ecosystem. Supporting all runtimes simultaneously is not an M1 goal. [NODE-1, JS-2, LUA-1]

The M1 decision record must compare the same command/list/form/streaming examples, compatibility gaps, startup, resource footprint, reload latency and fault containment. TypeScript requires bundling; compiled sidecars can rebuild/restart too. The desired property is **fast, reliable development feedback**, not “interpreted” as a proxy for either speed or reloadability.

## 7. Native extension UI contract

### 7.1 Responsibilities and flow

```text
TS source + manifest
    │ dev compiler / package validation
    ▼
extension helper: business state, providers, async handlers
    │ versioned requests + native tree transactions
    ▼
host protocol/service boundary: validate, authorize, cancel, bound queues
    │ immutable accepted model updates
    ▼
Go native adapter: pane/node state, layout, focus, input, viewport
    │ MyGo view builds; no synchronous extension call
    ▼
GPU/native platform surface
    └── semantic input events / requested visible ranges ──► helper
```

The UI thread always has a last accepted model to render. It never awaits an extension's `render`, row callback, condition, measurement, hover handler or RPC response. Decoding/schema validation and provider computation occur outside it; a bounded main-thread commit updates host-owned state and invalidates only when needed. MyGo's multiple builds per frame must not repeat external side effects. [MYGO-2, BLEND-3]

### 7.2 Component vocabulary

Start with `Row`, `Column`, constrained `Grid`, `Text`, `Icon`, `Image`, `Button`, `Toggle`, `TextInput`, `Select`, `Form`, `Progress`, `EmptyState`, `VirtualList`, `Tree`, `Table` and a limited Markdown/detail surface. Their controls share the same tokens and semantics as the shell. High-level `Terminal`/`Editor`/`Diff` surfaces reference host-managed sessions/documents; they do not serialize terminal cells or full editor buffers into the extension tree.

Layout properties are typed values: grow/shrink, min/max size, spacing tokens, alignment and overflow. Theme references are semantic tokens, with documented per-component overrides. Extensions can compose real custom panels; they cannot submit arbitrary native pointers, MyGo internals, shaders or an unrestricted drawing callback through this API. A later specialized canvas capability needs its own use case and performance contract.

**Why this level:** Blender's `UILayout` pairs semantic controls with host properties/operators; Raycast makes coherent lists/forms/actions available without exposing all AppKit. These are useful precedents for a small SDK. Their documented success is not evidence that Patch's particular component set is sufficient; the reference extensions must test that. [BLEND-1, BLEND-2, RAY-1, RAY-3]

### 7.3 Identity, lifecycle and state

- Every panel type has a durable contribution ID; every open pane has a distinct host ID. Every component has an explicit stable key within its panel tree. SDK-generated positional keys are permitted only for stateless fixed children.
- Wire node identity is `(extension ID, activation generation, pane ID, node ID)`. Logical component keys and state schema can map a replacement generation to compatible saved host state; old wire IDs never become valid again.
- **Same node type and key:** preserve focus eligibility, text editing state, selection and scroll anchor. **Changed type/key:** create a new control. **Removal:** dispose callbacks/resources, cancel node requests and move focus predictably.
- Pane move/resize is not unmount. Visibility is a lifecycle event; hiding pauses render production and unnecessary data work, while retaining small model/state according to policy.
- The Go adapter owns `ui.Handle`, `ListState`, text values and similar persistent state; it never retains `ui.Element` across builds. Because MyGo identity includes ancestry, docking must preserve/rebind host state explicitly when the split parent changes. [MYGO-2]
- Host state: placement, active tab, focus, native editing/undo state, scroll anchor and view preferences. Extension state: query, filters, chosen resource, business model and explicitly serialized draft. External service state: terminal process, document or task under its own owner.
- A UI tree is a view, not authoritative storage for unsaved documents. Secrets and process handles are never serialized into layout/reload snapshots.

### 7.4 Input, focus and text

The host handles pointer capture, drag thresholds, resize, scrolling physics, selection, hover/pressed feedback and immediate text entry locally. Extensions receive semantic events such as `activate`, `change`, `submit` and `selectionChanged`; high-frequency raw pointer events are not the default extension API.

Events include pane/node identity, activation generation and an input sequence. Text changes include an edit revision and composition state. A response based on an older revision cannot replace newer host text. Ordinary controlled-value echoes acknowledge an edit; they do not reset cursor/undo/IME. Explicit reset/replacement is a separate operation with a revision precondition.

`requestFocus` is asynchronous intent, not an unconditional action. The host applies it after the relevant tree commits, only to a present focusable control in the target pane. Opening a panel may request initial focus; background updates do not steal focus. On deletion, focus goes to the next appropriate control, pane header or neighboring pane. An overlay owns a focus scope and returns focus to its still-valid origin on dismissal. Cancel obsolete focus requests on hide/close/reload.

### 7.5 Keyed transactions and rendering

The initial transaction creates a tree. Later transactions use `insert`, `setProps`, `move` and `remove` against stable node IDs. Each transaction carries generation, pane, base revision and next revision; an accepted transaction is atomic. Unknown nodes, duplicate keys, cycles, invalid properties and stale revisions return structured errors without partially changing the visible tree.

Use **one unacknowledged transaction per pane** initially. While it is in flight, coalesce local state and compute the next diff from the last acknowledged tree. This avoids unbounded revision chains and makes a full snapshot resync understandable. Full snapshots are for initial mount/recovery, not every scroll tick. `committed` means accepted into host state; it does not promise pixels are on screen. An explicit diagnostic `presented` receipt may be used for latency measurement.

remote-dom provides useful mutation/receiver/batching and callback-lifetime precedents, but its DOM/JS adapters are not an existing MyGo integration. A simple typed builder comes first; React/JSX can later emit the same protocol without changing native semantics. [REMOTE-1, REMOTE-2]

### 7.6 Virtual collections

A virtual list/tree/table references a registered data source, count or unknown-count cursor, dataset revision, row-key field and a **host-renderable row template**. It does not serialize the full dataset or call extension JS to build each visible row during a frame.

The host requests a visible index range plus bounded overscan asynchronously. Responses have query epoch, dataset revision, range and stable item keys. Moving the viewport cancels obsolete requests; out-of-order responses for old epochs are discarded. Pending rows show stable placeholders and retryable errors. Host cache keys include source/revision/item; scroll anchoring uses an item key plus intra-row offset. Selection follows item keys across sort/insert, with explicit behavior when a selected item disappears.

Start with one page ahead/behind and a bounded cache; measure before increasing it. Custom variable-height rows report native measured height into the host's layout model; they do not require an IPC round trip for every measurement. Large datasets remain large data, not large native trees. [MYGO-5, BLEND-3]

## 8. Proposed authoring experience and examples

**Everything in this section is illustrative future SDK/schema design. It is not the M0 API and is not runnable against a published Patch package.** Naming should be refined through M1 rather than silently treated as frozen compatibility.

### 8.1 Development loop

```sh
patch extension init acme.project-files --template native-list
patch extension dev ./project-files
patch extension inspect acme.project-files
patch extension pack ./project-files
patch extension install ./acme.project-files-0.1.0.patch-extension
```

`dev` watches sources, typechecks/bundles, validates the manifest and atomically selects a compiled generation. The native preview is a real Patch pane using the production adapter. Diagnostics show the original TS source location, activation reason, render/IPC timings and owned registrations. Console output appears in an extension-specific log, with no need to expose a network debugger by default.

Provide a native tree inspector with keys, resolved tokens, accessibility names, focus, bounds and last transaction. A REPL scoped to the selected extension is desirable after lifecycle basics. Full step debugging is a runtime selection criterion; it is not promised by simply adding source maps to goja.

### 8.2 Manifest example

```json
{
  "schemaVersion": 1,
  "id": "acme.project-files",
  "name": "Project Files",
  "version": "0.1.0",
  "engines": { "patch": ">=0.2.0 <0.3.0", "api": "1" },
  "runtime": { "kind": "javascript", "entry": "dist/extension.js" },
  "permissions": [
    {
      "capability": "workspace.read",
      "reason": "Show and filter files in the open workspace"
    }
  ],
  "contributes": {
    "commands": [
      { "id": "acme.project-files.open", "title": "Open Project Files" }
    ],
    "panels": [
      {
        "id": "acme.project-files.panel",
        "title": "Project Files",
        "surface": "native",
        "instances": "multiple",
        "defaultPlacement": "left"
      }
    ],
    "settings": {
      "acme.project-files.showHidden": {
        "type": "boolean",
        "default": false,
        "scope": ["user", "workspace"],
        "description": "Include hidden files"
      }
    }
  }
}
```

Opening the declared command or panel infers activation. `defaultPlacement` is a hint for first creation, never permission to rearrange a user's saved layout. Numeric app/API versions above illustrate compatibility ranges; they are not release dates or an assertion of a released Patch 0.2.

### 8.3 Native list authoring example

The proposed SDK hides callback IDs and tree diffing. Data sources are asynchronous and are disposed with the pane; the host draws rows from a declared template.

```ts
import { defineExtension, ui } from "@patch/sdk";

export default defineExtension({
  activate(ctx) {
    ctx.commands.register("acme.project-files.open", async () => {
      await ctx.panels.open("acme.project-files.panel");
    });

    ctx.panels.register("acme.project-files.panel", (panel) => {
      let query = panel.state.get<string>("query", "");

      const files = panel.dataSources.register("files", {
        async getRows({ start, count, signal }) {
          // Host returns { revision, total, items }, with stable item keys.
          // Service failures become a retryable list error; abort is silent.
          return ctx.workspace.listFiles({
            query,
            start,
            count,
            showHidden: ctx.settings.get("acme.project-files.showHidden"),
            signal,
          });
        },
      });

      function render() {
        panel.render(ui.column({ key: "root", gap: "space.2" }, [
          ui.textInput({
            key: "query",
            label: "Filter files",
            value: query,
            onChange(event) {
              query = event.value;
              panel.state.set("query", query);
              files.invalidate(); // Cancels old ranges and advances query epoch.
              render();
            },
          }),
          ui.virtualList({
            key: "files",
            source: files,
            row: { key: "uri", title: "name", subtitle: "relativePath" },
            onActivate: async ({ item }) => {
              await ctx.commands.execute("patch.openResource", { uri: item.uri });
            },
          }),
        ]));
      }

      panel.subscriptions.add(ctx.settings.onChange(
        "acme.project-files.showHidden", () => files.invalidate(),
      ));
      render();
    });
  },
});
```

All registrations attach to the current activation or pane scope. SDK event dispatch catches rejected async handlers and routes them to pane diagnostics; data-source failure displays retry instead of swallowing the error or returning an empty successful list. `files.invalidate()` cancels obsolete work; a query captured by an in-flight request is paired with its original epoch. Settings/defaults are typed from the manifest by code generation.

### 8.4 Representative wire transaction

```json
{
  "protocol": 1,
  "type": "ui.commit",
  "extension": "acme.project-files",
  "generation": 7,
  "pane": "pane-42",
  "baseRevision": 11,
  "revision": 12,
  "operations": [
    {
      "op": "setProps",
      "node": "query",
      "props": { "value": "readme", "ackEditRevision": 24 }
    },
    {
      "op": "setProps",
      "node": "files",
      "props": { "dataRevision": "query-9" }
    }
  ]
}
```

The receiving endpoint already knows the authenticated extension/process identity and verifies these fields; a message cannot impersonate another extension merely by changing `extension`. Functions, pointers and native handles never cross this boundary. An event names a registered callback token plus generation, node, event sequence and typed payload.

### 8.5 Schema fragment

This illustrative JSON Schema constrains an individual property-update operation. The full protocol schema must additionally define the component-property union and enforce tree/revision invariants at runtime; JSON shape validation alone cannot prove them.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "PatchSetPropsOperation",
  "type": "object",
  "additionalProperties": false,
  "required": ["op", "node", "props"],
  "properties": {
    "op": { "const": "setProps" },
    "node": { "type": "string", "minLength": 1, "maxLength": 128 },
    "props": { "type": "object", "maxProperties": 64 }
  }
}
```

## 9. Async protocol, capacity and cancellation

### Transport

Use framed JSON over inherited stdin/stdout pipes for the first local helper. Each frame has a byte length and UTF-8 body; stderr is diagnostic output, never protocol data. JSON is inspectable and has native Go support. A binary codec or shared memory requires measured need. Large file/process streams use bounded chunks/resource handles instead of embedding whole data in UI transactions.

Handshake: protocol major/minor, host/API version, extension identity/version, negotiated feature set, supported component versions and limits. Unknown major versions fail before activation. Optional fields/features can be negotiated; unknown mandatory operations fail rather than silently losing behavior.

### Request contract

Requests have an ID, method, scope, generation, deadline and typed arguments. Responses are success or structured errors (`code`, safe message, retryability, request ID). Cancellation is explicit and idempotent; it cancels host I/O/subprocess work where supported, not merely the promise waiting for it. Scope disposal aborts its outstanding requests and subscriptions. Late replies after cancellation/close/reload are discarded.

One process exit rejects all its pending requests. The UI retains its last accepted state and reports the affected extension, not an unrelated generic application failure. No synchronous RPC is offered to extension UI handlers.

### Backpressure policy

- **UI trees:** one in-flight transaction per pane, latest pending model coalesced as described above. Never drop arbitrary diffs from a revision chain.
- **Lossless results/byte streams:** credit-based bounded chunks; when credit is exhausted, producers await it or receive a defined overflow error. Do not silently truncate process output while presenting it as complete.
- **Replaceable state:** coalesce progress, viewport intent and status by key. Count/report coalescing in diagnostics.
- **Cancellation/control:** reserve queue capacity so a flooded data stream cannot prevent cancellation or shutdown.
- **Scheduling:** fair queues across active panes/extensions; bounded commits per frame. Oversized transactions are rejected or prepared off-thread before atomic adoption, not partly painted.
- **Limits:** negotiate maximum frame size, queued bytes, concurrent requests and snapshot size. The starting numbers in §13 are trial budgets, not measured safe defaults.

For goja, `Interrupt` addresses running JS; exposed Go functions must not block the event loop awaiting the UI and must implement context cancellation. A runaway helper can be terminated independently. A process watchdog is a recovery mechanism, not an assertion of hard real-time scheduling or hard memory isolation. [JS-1]

## 10. Configuration, themes, keybindings and layouts

### 10.1 Declarative configuration

Use versioned, schema-validated JSON for the initial configuration language. A settings UI edits the same model. Keep declarative configuration separate from optional executable customization so loading a project layout does not implicitly execute code.

**Proposed resolution, low → high:** core defaults → installed extension defaults in their own namespaces → selected user profile → user overrides → workspace overrides for workspace-scoped settings → session overrides. A pane-instance override applies only to settings declared pane-scoped and is highest within that pane. Language/resource selectors, if later added, require separately documented precedence rather than implicit exceptions.

Merge rules: recursively merge declared objects; scalar/array values replace unless a schema explicitly supplies keyed-merge behavior; omission inherits; `null` is only a value when allowed by the schema. “Reset” removes an override. Show the winning value, source and overridden values via **Explain Setting**. Permissions, executable paths and global runtime management are user-scoped; a repository cannot grant itself authority. VS Code is precedent for explicit settings scopes, not a requirement to copy every precedence tier. [VSC-6]

Valid edits apply atomically; schema errors preserve the last valid settings and identify the field/location. A watcher handles external edits and avoids save loops. Mark settings that need pane recreation, extension restart or app restart; apply cheap changes such as tokens in place. Report failed persistence and keep user edits recoverable.

### 10.2 Themes

Tokens cover surface layers, text roles, separators, accent, focus, selection, semantic status, spacing/density, typography and radii. A theme can inherit a base light/dark theme and override tokens. Component authors use semantic roles; user customizations can override theme tokens. Validate token types and cycles; surface missing tokens using host fallbacks. Font size/contrast/reduced-motion settings apply consistently to shell and native extension panels.

Extensions may contribute theme data with no runtime. Webviews receive a documented token snapshot/change message when enabled, but cannot override host chrome. Full arbitrary CSS for the shell would couple extensions to renderer internals and is not part of the native API. [VSC-4, RAY-3]

### 10.3 Keybindings

Bindings refer to command IDs, typed arguments and a small host-evaluated `when` expression grammar: booleans, equality, negation and conjunction/disjunction over documented context keys. No JavaScript predicate runs on keypress. Context includes focused pane type, text-input focus, terminal focus, active workspace and selection availability.

Resolve by explicit layer priority, then order within that layer; show conflicts and the exact winning rule. Chords have a visible pending state, timeout and cancel behavior. User removal rules disable a specific binding rather than depending on extension load order. Respect terminal/editing keys and international keyboard layouts; platform defaults must be tested. GoRex explicitly preserves terminal Control-letter shortcuts outside macOS and avoids AltGr conflicts. [VSC-6, REF-2]

### 10.4 Layout presets and session state

Presets are templates, not the live session file. Applying a preset is explicit and reversible; it maps named pane roles to new/existing instances with a preview of affected placement. Normal settings reload never silently rearranges an active workspace. Store local workspace session state outside tracked project files by default; export/share a preset deliberately. Do not serialize secrets, process environment or arbitrary absolute paths into a shared preset.

Illustrative future layout model:

```json
{
  "schemaVersion": 1,
  "workspace": "workspace-1",
  "activePane": "terminal-1",
  "root": {
    "kind": "split",
    "id": "split-main",
    "axis": "horizontal",
    "ratio": 0.28,
    "first": {
      "kind": "group",
      "id": "group-files",
      "panes": ["files-1"],
      "active": "files-1"
    },
    "second": {
      "kind": "group",
      "id": "group-work",
      "panes": ["terminal-1", "review-1"],
      "active": "terminal-1"
    }
  },
  "panes": {
    "files-1": { "type": "patch.project", "state": {} },
    "terminal-1": { "type": "patch.terminal", "state": { "cwd": "workspace:/" } },
    "review-1": { "type": "acme.review.panel", "state": { "schemaVersion": 1 } }
  }
}
```

Here horizontal means children are side by side. Ratios are preferences, not fixed pixel sizes. This proposed example illustrates a separate descriptor map and a multi-panel group; neither mandates M0 serialization or a new grouped-panel UX. M0 presents `State.Workspaces` through either Sidebar or title-bar Tabs and embeds panel descriptors within group nodes. Existing multi-panel groups use a compact header selector; new panels open visible panes. Its actual persisted format remains defined by the implementation docs.

## 11. Capabilities and optional webviews

### Pragmatic permissions

The problem to solve is concrete: extensions can read project files, contact services or run tools. Declare those needs with a short reason; show them when enabling/installing an extension and when an update adds authority. Remember user choices. A denied operation returns a typed error the panel can explain; ordinary authorized operations do not repeatedly prompt.

Initial capability vocabulary: `workspace.read`, `workspace.write`, `network.fetch` with allowed origins, `process.spawn`, `clipboard.read/write`, extension-scoped `storage`, and `webview`. Host services enforce the approved resource scope and caller identity. Resolve file scope against canonical resource paths; a string prefix alone must not grant access outside a workspace. Redirects follow network scope checks. Revocation cancels affected operations and removes access on the next request.

No marketplace review board or enterprise policy service is required for local development. A dev extension still has an explicit enabled identity and capabilities. Workspace files cannot grant their own permissions. Extension-private storage and platform secret storage are separate APIs; secrets do not enter logs, themes, layout exports or reload snapshots.

**Boundary honesty:** an embedded interpreter with restricted host imports is not automatically an audited sandbox. Per-extension helper processes provide failure isolation, not OS-level file/network confinement. A native executable or unrestricted Node runtime can exercise the user's OS authority outside Patch RPC; label that as trusted-code execution rather than promising that a manifest contains it. `process.spawn` can itself grant broad authority; a command allowlist is not a general sandbox. [RAY-2, ZED-2, GO-3]

### Webview extension surface

- Opt-in manifest `surface: "webview"` plus the `webview` capability; instantiate only when the user opens that pane.
- Use MyGo's untrusted-page mode for extension/third-party content so the ordinary privileged application bridge is unavailable. Expose only a dedicated validated extension bridge scoped to that pane/generation and the extension's approved services.
- Host retains pane header, docking, window controls, focus boundary and lifecycle. The page cannot impersonate another extension or call arbitrary bound application services.
- Packaged HTML uses an explicit CSP/resource policy. External navigation is explicit; failed navigation and missing platform webview runtime produce a recoverable pane state.
- Small JSON state can restore a suspended page. Default future extension policy should release hidden pages after a measured idle interval unless retention is requested and visible to the user. MyGo hides and retains by default, so this requires explicit host work. Closing a pane destroys its page.
- Test focus return, native overlays and capture using the actual native system. MyGo's host capture omits the embedded page; composite OS capture or separate page capture is required evidence. [MYGO-6, VSC-5]

The M0 optional webview is a framework/shell demonstration. It is not an implemented secure extension bridge or proof of hidden-page resource policy.

## 12. Reload, recovery and packaging

### Reload lifecycle

1. Compile/typecheck and validate the new package before disturbing the active extension. On failure, leave the old version running and show source diagnostics.
2. Quiesce the old activation: stop dispatching new handlers, cancel in-flight requests and ask for a small versioned snapshot with a deadline. The host already owns layout/focus/scroll; it need not ask the extension to serialize them.
3. Revoke the old generation's callback/registration scope and terminate/release its helper. Keep the last accepted native tree as a noninteractive loading/recovery view; do not dispatch its old callbacks.
4. Start a fresh helper, activate once, restore compatible state and publish a validated initial tree. Rebind compatible keyed native controls, restoring focus only if the user has not moved it elsewhere.
5. If activation/restore fails, show retry and load-last-good actions. A last-good package can start in a fresh helper from the saved state. Do not claim to roll back already completed external side effects or recover arbitrary VM heap/closures.

Schemas can provide an explicit state migration; unsupported state falls back with a clear notice and retained recoverable draft data when feasible. Reloading a list should keep selection/anchor by key, not by old numeric index. Native IME composition may require deferring a swap until the current composition commits/cancels; test it rather than force-blurring the control.

Host resources use explicit ownership. Event subscriptions, timers, callbacks, watchers and owned jobs are released on scope disposal. A host-managed terminal/document can persist across panel reload according to its separate lifecycle. Reload must not silently rerun an external command or launch a duplicate process simply because activation happened again. Blender's register/unregister and IntelliJ's disposal restrictions illustrate why cleanup is part of the extension API. [BLEND-1, IDEA-2, IDEA-3]

### Error recovery

Display the failed extension and stage: load, activation, service, render validation or process exit. Offer reload, disable and logs without restarting the whole app. User-owned documents/terminals stay available. Suppress automatic crash loops; after a repeated automatic restart, stop and ask for an explicit retry. Include request/generation identifiers in diagnostics, with safe error messages and source-mapped developer details.

### Versioning and packages

Separate package semantic version, manifest schema, host API major/minor, wire protocol version, component feature versions and serialized state version. M1/M2 APIs remain explicitly experimental; compatibility guarantees begin when a version is published. Negotiate additive features; do not silently render an unknown required control as an empty box.

A local `.patch-extension` archive contains the manifest, JS bundle/source map, static assets, licenses and optional declared platform binaries. Validation prevents archive path traversal and verifies declared files, sizes and hashes. A local manifest's claims are not proof of trusted authorship. No dependency installation or arbitrary package lifecycle scripts run merely by opening a workspace.

Prefer bundled pure-JS dependencies to runtime package resolution. A later sidecar package declares OS/architecture, checksum and protocol support. Go shared-library plugins are unsuitable for portable unload/reload; use compiled helpers over the same async protocol. QuickJS bytecode is runtime-version-specific and should not be the distributable compatibility surface. [GO-1, JS-2, BLEND-6]

Install/update stages a complete package, validates compatibility and permissions, then swaps the selected version at a lifecycle boundary. Retain the last good package for rollback. Uninstall unregisters its contributions and leaves recoverable pane placeholders. M4 can add a simple signed/indexed distribution channel without making a marketplace a prerequisite for useful local extensions.

## 13. Performance targets and measurement plan

**Every budget in the table below is a proposed engineering target, pending measurement and revision after M0/M1.** They are neither shipped guarantees nor borrowed upstream benchmark results. The separately cited M0 diagnostic at the end of this section is not a budget result. Record deviations honestly; a failed budget should inform the design, not be hidden by narrowing the fixture.

### Reference workload

Select and record one development-machine baseline: hardware/CPU/RAM, OS, display scale/refresh rate, release build, exact Go/MyGo/runtime versions. Use a 60 Hz baseline plus a high-refresh check where available. Separate cold start from warm-cache start and exclude compile/download time from application startup; include compile/bundle time in edit-to-feedback DX.

Fixtures: empty shell; one workspace with four native panes including an idle terminal; 10 installed/0 active extensions; 10 active lightweight panels; a 100,000-item provider with only viewport data materialized; one fast stream and one stalled extension. Measure webview processes separately and in the whole process tree.

| Metric | Initial target | Measurement / interpretation |
|---|---|---|
| Cold shell to first usable frame | p95 ≤ 500 ms | Process launch to first presented interactive native frame on the declared baseline; no webviews/extensions auto-started. |
| Warm shell to first usable frame | p95 ≤ 250 ms | Same build/workspace and warm filesystem cache; report separately. |
| Host input-to-present during resize/scroll | p95 ≤ 16.7 ms, p99 ≤ 33.3 ms at 60 Hz | Native event timestamp to presented frame, not handler completion; record any dropped frames. |
| Native UI thread work | p95 ≤ 4 ms/frame | Build/layout/commit CPU work, reported separately from GPU/presentation and OS scheduling. |
| Extension UI commit work on main thread | p95 ≤ 1 ms for a 200-visible-node fixture | Decode/validation preparation excluded from UI thread, but report full request-to-visible latency too. |
| Lightweight command/panel activation | p95 ≤ 100 ms / 200 ms | Already-installed package; separate helper startup, script initialization, data fetch and first native paint. |
| TS save to updated native pane | p95 ≤ 300 ms warm, ≤ 1 s cold | Includes bundle/typecheck selection, helper replacement, state restoration and first paint for reference extension. |
| Idle shell + four native panes | ≤ 150 MiB whole app/helper resident footprint | Platform metric documented; exclude user child shell applications only in a separately labeled subtotal. Report GPU memory and child processes; no direct RSS↔physical-footprint equivalence. |
| Lightweight active helper | ≤ 20 MiB incremental median | Candidate comparison, not a reason to misrepresent Node/goja limitations; include 10-helper aggregate. |
| Idle CPU | < 0.5% of one core over 30 s | Settled UI, no active jobs/animations; background native/OS wakeups included. |
| Bounded native list | ≤ 3 viewport pages + focused editing row resident initially | Inspect native nodes and transfer volume with 100,000 records; dataset metadata can be larger, rendered rows cannot scale linearly. |
| IPC trial limits | 256 KiB/frame; 1 MiB queued data per extension; 32 outstanding service requests | Configurable negotiated starting limits; verify cancellation/control remains deliverable at saturation. Chunk larger data. |
| State snapshots | ≤ 1 MiB/extension; 250 ms snapshot deadline | Reject/diagnose oversized state; retain host state and last valid snapshot without waiting forever. |
| Cancellation feedback | UI acknowledgment ≤ 50 ms; cooperative task stops ≤ 250 ms | Differentiate acknowledgment, cancellation request and actual task termination. Kill stubborn helper separately with diagnostic. |

Use at least 30 startup/reload trials for percentile claims; retain sample counts and outliers. For memory/idle work, report steady-state snapshots and repeated reload trends rather than one favorable observation. Capture build/layout/input/IPC timestamps and workload sizes. A source-level unit test can prove tree invariants; it cannot prove native focus, visual quality, presentation latency or GPU footprint.

MyGo publishes useful headless CPU fixture numbers, but they exclude native-window/GPU resources and are not a Patch baseline. [MYGO-9] M0 recorded one main-process RSS snapshot of **96,240 KiB (about 94 MiB)** with three panes and the browser closed. It excludes shells, WebKit helpers and GPU physical footprint; it is a diagnostic observation, **not a whole-application budget pass**. Startup budgets, input-latency percentiles and idle CPU remain unmeasured. See [measurements](verification.md#measurements); the bounded runtime comparison belongs to M1.

## 14. Acceptance criteria

### M0 — current initial-shell acceptance

**This A0 table records M0 acceptance scope with build-specific outcomes.** Above/Left/Right behavior, terminal sizing, pill states and native Changes now have final [refinement acceptance](verification.md#refinement-batch-tab-style-diff-review-glyph-size). Runtime evidence is macOS Apple silicon only, with popup-action limitations retained below. Background extension moves, rename, live configuration reload and missing-extension recovery remain later milestones; a richer grouped-panel UX remains a product decision.

Evidence provenance: the [refinement report](verification.md#refinement-batch-tab-style-diff-review-glyph-size) records final debug `a1fa5260…` and release `50161696…` hashes, each observation's build and reused unchanged-area evidence. Final native acceptance is PASS, and the coordinator reviewed the current diff, Above and Right Side captures. This documentation task did not run or independently certify those journeys.

| ID | Required observation | Scoped outcome / evidence |
|---|---|---|
| A0-1 | Native launch uses the pinned dependency/toolchain and starts no webview by default. No-argument startup, including the Finder-launch case, restores a valid workspace or offers folder selection. | **Verified on macOS Apple silicon:** fresh no-argument launch from `/` shows Open a project and saves no workspace; restart restoration verified separately. No webview/request before explicit Go. [Report](verification.md#verified) |
| A0-2 | New panels are visible panes; nested splits, edge docking, center swaps, resizing and zoom/restore preserve identity/session state. Legacy groups remain reachable without extra tab rows. | **Native-verified:** pane IDs retained; typed commands confirmed the same shell PID/TTY after docking, swaps, resizing and workspace changes. Pointer/keyboard split changes and reset passed. Every legacy panel remains reachable through the header selector/menu. [Report](verification.md#verified) |
| A0-3 | Save/restart restores descriptors, order, ratios and settings; invalid saved data is handled without destroying the last valid data. | **Native restart verified:** descriptors/IDs/order, ratios, navigation/appearance, window size and geometry restored. Invalid saved-file handling is covered by **core tests**, as reported by the coordinator; no native corrupt-file journey is claimed. [Native evidence](verification.md#verified) |
| A0-4 | Above/Left/Right supplies one hierarchy; mode/side/visibility persist independently; sidebar toggle only changes visibility in Sidebar mode; no automatic viewport mode switch. | **Native PASS:** requested modes/radio states, restart persistence, independent visibility, 900×700 mode retention, aligned hover-close X, whole-pill focus/pressed states and mirrored Right Side inset. [Refinement evidence](verification.md#verified-natively) |
| A0-5 | Terminal/files/native Git diff are useful; non-Git state is friendly; icon sizing matches the chosen reference; gallery is singleton; optional browser loads/closes explicitly. | **Native PASS within scope:** accepted Kero glyph sizing/ink overhang, Files/Changes corners, readable real-line EOF diffs and consistent word colors. Popup Unified/Split/Collapse all/Expand all actions have source/headless evidence only. Browser lifecycle has beacon evidence; embedded-page pixels and Windows runtime remain unverified. [Report](verification.md#refinement-batch-tab-style-diff-review-glyph-size) |
| A0-6 | Compare current chrome with Kero/GoRex and native diff presentation with Godiff; assess focus, pressed states, resize and content geometry. | **Final native refinement PASS**, following earlier Opus 5.5 xhigh reference approval. The coordinator inspected current diff, Above and Right Side captures. [Current screenshots](verification.md#current-screenshots) |
| A0-7 | Record measurement scope, actual platform coverage and future extension features accurately. | **Documented:** final binaries, macOS Apple-silicon environment, one ~94 MiB main-process RSS diagnostic and GPU frame logs. No startup/input/idle-CPU budget pass is claimed; the extension runtime/SDK remain future proposals. [Measurements](verification.md#measurements) |

**Verification limits:** only macOS Apple silicon was exercised (Apple M2, macOS 26.6.2). Popup Unified/Split/Collapse all/Expand all actions were inaccessible to native tooling and retain source/headless coverage only. Screen-recording permission prevented OS traffic-light/webview pixel capture; browser lifecycle has beacon evidence. Refinement checks include 900×700, but no full viewport matrix or 1440×900 content capture is claimed. Invalid saved-file behavior has core-test rather than native-corruption evidence. M0 ends its shells on quit; daemon persistence is outside A0-2/A0-3. [Refinement limits](verification.md#not-verified-natively)

### Future extension acceptance

| ID | Required observation | Milestone |
|---|---|---|
| A1-1 | Same reference extensions run through candidate runtimes; record JS/API compatibility, inspector/DX, CPU/memory/start/reload costs and selected runtime rationale. | M1 |
| A1-2 | A busy or infinite-loop extension cannot block shell typing, scrolling or resizing; helper termination leaves other panes usable. | M1/M2 |
| A1-3 | Install manifests without activation; command/panel/service demand activates once; all registrations disappear when their owner scope ends. | M2 |
| A1-4 | Keyed data/tree changes preserve native selection, edit/IME and focus; docking a pane preserves state despite MyGo parent identity changes. Removing focused content has a deterministic fallback. | M1/M2 |
| A1-5 | The 100,000-item list requests bounded ranges, cancels obsolete epochs, ignores late data, keeps anchors on insertion and remains scrollable with a slow provider. | M1/M2 |
| A1-6 | A flood respects queue/request limits; cancellation and shutdown remain responsive; lossless streams report overflow rather than silently losing data. | M2 |
| A1-7 | Compile failure preserves the running generation; activation failure offers last-good restart; state mismatch is explained; repeated reload leaves no duplicate watchers/jobs/callbacks. | M2 |
| A1-8 | A denied/revoked file/network capability yields a useful error; workspace config cannot grant it; native/Node trusted-code authority is disclosed accurately. | M2 |
| A1-9 | Explain Setting identifies the winning layer; UI/file edits round-trip; theme/keybinding/layout contributions work without activating code; applying a preset is explicit and recoverable. | M3 |
| A1-10 | A local package can install, update, fail validation, roll back and uninstall while unrelated panes continue; unsupported API/component versions fail clearly. | M2/M4 |
| A1-11 | Webview is created only on demand, has only its dedicated approved bridge, can fail independently, returns keyboard focus and obeys documented close/suspend policy. | M3 |
| A1-12 | An author unfamiliar with host internals builds a useful native list/form/command extension from the published example and diagnoses one failure using tools/docs. | M2/M3 |

## 15. Open decisions and risks

1. **Runtime:** goja is the first candidate, not a final compatibility commitment. Decide using real extension dependencies and DX as well as resource data. Avoid creating a partial Node clone.
2. **MyGo pin and build evidence:** `go.mod` pins the researched revision as `v0.0.0-20261103171621-6842ad5e0aa2`, with Go 1.27.3. Debug/release builds and scoped macOS native acceptance are recorded in [verification.md](verification.md). The earlier Go 1.24.2 observation is historical. The framework review identified relevant drag/webview fixes after v0.7.4, motivating the exact revision.
3. **Toolkit behavior:** M0 has native keyboard/focus and terminal-continuity evidence. IME breadth, full assistive-technology behavior, OS-composited webview pixels and other platform behavior still require their own evidence. Do not extrapolate current accessibility-tree automation into a complete screen-reader acceptance result or replace native panels to hide gaps.
4. **Panel vocabulary:** determine sufficiency through file search, task monitor, review details and a small settings form. A broad canvas/React adapter should follow a concrete missing use case.
5. **Isolation versus footprint:** one helper per active extension is simple, but its aggregate cost is unknown. Pooling can be considered after measurement; it changes failure/accounting boundaries and must be a deliberate trade-off.
6. **Session continuity:** local terminal state survives moves/hiding; process continuity across app exit/remote disconnect needs a session service. Kero/GoRex document daemons; MyGo's terminal widget alone does not provide that product contract. Its libghostty-vt integration loads a bundled native library: Go-only application code and no cgo do not mean native-library-free distribution. [REF-1, REF-2, MYGO-8]
7. **Platform promise:** native runtime verification is macOS Apple silicon only. A **Windows x64 portable package has been built and its PE executable, DLL and archive inspected**; this is packaging evidence, not native Windows launch/rendering/input verification. A GitHub Actions native-Windows build is configured for the forthcoming push to `main`; no CI success is claimed before that job runs. See [Windows packaging](windows.md) and the [runtime report](verification.md). Linux and other macOS architectures remain unverified.

## 16. Rationale summary and sources

The recommendation combines **Blender's host-native panel/property/operator model**, **Raycast's TS-to-native rendering boundary**, **VS Code's declarative contributions and activation**, and **Neovim's explicit scheduling/versioned API discipline**. Zed demonstrates a useful WASM/provider model; Qt and Blender document why cheap declarative syntax still requires short render-path work. Sublime and IntelliJ inform lifecycle and failure isolation. These are targeted lessons, not a requirement to reproduce their entire APIs.

Detailed comparisons, runtime alternatives, source dates/revisions and limitations: [research.md](research.md). Primary entry points:

- [MyGo native views](https://github.com/egoist/mygo/blob/6842ad5e0aa2cd168166e0a4ae12c0cd0f527a3d/docs/ui/views.md)
- [Blender Panel](https://docs.blender.org/api/current/bpy.types.Panel.html), [UILayout](https://docs.blender.org/api/current/bpy.types.UILayout.html), [areas](https://docs.blender.org/manual/en/latest/interface/window_system/areas.html)
- [Raycast native UI FAQ](https://developers.raycast.com/misc/faq) and [runtime model](https://developers.raycast.com/information/security)
- [VS Code extension host](https://code.visualstudio.com/api/advanced-topics/extension-host), [contributions](https://code.visualstudio.com/api/references/contribution-points), [webviews](https://code.visualstudio.com/api/extension-guides/webview)
- [Neovim API](https://neovim.io/doc/user/api.html), [Zed extensions](https://zed.dev/docs/extensions/developing-extensions)
- [goja](https://github.com/dop251/goja), [QuickJS](https://bellard.org/quickjs/quickjs.html), [Node Inspector](https://nodejs.org/api/inspector.html)
