# Bundled symbol fallback

`JetBrainsMonoNerdFont-Regular.ttf` is the unchanged non-Mono **JetBrainsMono Nerd Font Regular** asset from Nerd Fonts v3.5.1 at commit `b894ea7803af6aade63d60a4381e006098ec9c4d`. It is used only as a symbol fallback, not the primary terminal font.

- Source: <https://github.com/ryanoasis/nerd-fonts/blob/b894ea7803af6aade63d60a4381e006098ec9c4d/patched-fonts/JetBrainsMono/Ligatures/JetBrainsMonoNerdFont-Regular.ttf>
- Size: 2,571,596 bytes
- SHA-256: `1c680e8cde9fcf8b88a5605ce8d1fb94dd3fb15841f7ca7bf4c55664855e5611`
- Runtime family: `JetBrainsMono Nerd Font`

Patch registers this font in-process as a fallback after Menlo on macOS, or the system monospace font elsewhere. It does not install a system font or modify shell/prompt configuration. ASCII uses the primary monospace font; supported Nerd Font glyphs use this fallback. This does not claim coverage of every private-use Unicode character.

A native Core Text probe established Menlo as the macOS primary that resolves both BMP and supplementary private-use symbols correctly. The former Symbols Nerd Font Mono fallback had a 13-point advance against Menlo's 8-point cell, causing mygo to shrink its ink to 8/13. This fallback's 7.8-point advance avoids that shrink at the unchanged 13-point font size. ASCII regular/bold/italic selection remains Menlo. Native inspection passed the actual spaced prompt's size, baseline and Powerline alignment. Wide ink can overlap tightly adjacent non-space characters; retaining this larger Kero-style sizing is the explicitly selected trade-off. No prompt or engine patch is applied. The exact Apple system-monospace fallback policy was not established.

`OFL.txt`, `NERD-FONTS-LICENSE` and `UPSTREAM-README.md` are unchanged upstream files from the same commit. JetBrains Mono is licensed under OFL 1.1. The patched font also combines icon sets; the included upstream README records their authors/sources and licenses, including MIT, CC BY 4.0, Apache 2.0, OFL 1.1 and Unlicense terms.
