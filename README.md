# Patch

A desktop IDE built with [MyGo](https://github.com/egoist/mygo), with a Go
backend and a TypeScript/Vite frontend.

## Prerequisites

- Go 1.27.3 or later
- [Bun](https://bun.sh)
- On Linux: GTK 3 and WebKitGTK 4.1 (for example,
  `sudo apt install libwebkit2gtk-4.1-0`)
- On Windows 10: the WebView2 Runtime (included with Windows 11)

## Getting started

```sh
bun install
bun run dev
```

To package the app for the current platform:

```sh
bun run build
```