package panels

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

type diffFile struct {
	diffSource
	diff           *diffDocument
	added, deleted int
	note           string
	limited        bool
	collapsed      bool
}

func (f *diffFile) key() string { return f.scope + ":" + f.path }

func loadDiffFile(ctx context.Context, source diffSource, remaining *int, read func(diffVersion, int) (string, error)) *diffFile {
	f := &diffFile{diffSource: source}
	if source.conflict {
		f.note = "This file has unresolved merge conflicts. Its index has multiple versions; resolve it in your editor to review a two-way diff."
		return f
	}
	if source.before.mode == "160000" || source.after.mode == "160000" {
		f.note = "Submodule changed. Open its folder to review the changes inside it."
		return f
	}
	before, err := read(source.before, min(diffFileLimit, *remaining))
	var after string
	if err == nil {
		after, err = read(source.after, min(diffFileLimit, *remaining-len(before)))
	}
	if err != nil {
		f.limited = errors.Is(err, errDiffLimit)
		f.note = err.Error()
		if !f.limited && !errors.Is(err, errDiffBinary) {
			f.note = "Couldn't read this change. " + f.note
		}
		return f
	}
	if before == after {
		f.note = "File contents are unchanged."
		if before == "" {
			f.note = "Empty file."
		}
	} else {
		f.diff, err = newDiffDocument(ctx, source.path, before, after)
		if err != nil {
			f.note = "Couldn't prepare this change. " + err.Error()
			return f
		}
		// Only a successfully retained pair uses the document budget. A binary
		// version, failed second read, or metadata-only change retains no text.
		*remaining -= len(before) + len(after)
		f.added, f.deleted = f.diff.added, f.diff.deleted
		if source.scope == "Staged" && strings.Contains(before, "\r\n") != strings.Contains(after, "\r\n") && before != "" && after != "" {
			f.note = "Line endings changed."
		}
	}
	f.note = strings.TrimSpace(f.note + " " + diffEOFNote(before, after))
	if a, b := source.before.mode, source.after.mode; a != "" && b != "" && a != "000000" && b != "000000" && a != b {
		f.note = strings.TrimSpace(f.note + fmt.Sprintf(" Mode %s → %s.", a, b))
	}
	return f
}

// raw preserves the source exactly. display removes only the terminal line
// ending; an editor's final caret row then represents the last real source line.
type diffText struct {
	raw, display string
	lines        []string
	finalLF      bool
}

func newDiffText(raw string) diffText {
	t := diffText{raw: raw, finalLF: strings.HasSuffix(raw, "\n")}
	if raw != "" {
		t.display = strings.TrimSuffix(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
		t.lines = strings.Split(t.display, "\n")
	}
	return t
}

// The widgets need different representations for zero logical lines: a native
// editor always has a caret row, so an absent side never gets its own editor.
type diffDocument struct {
	before, after  diffText
	comparison     *editor.DiffState
	single         *editor.State
	added, deleted int
}

func newDiffDocument(ctx context.Context, path, before, after string) (*diffDocument, error) {
	d := &diffDocument{before: newDiffText(before), after: newDiffText(after)}
	font := ui.Font{Family: "Menlo, monospace", Size: 12}
	if len(d.before.lines) == 0 || len(d.after.lines) == 0 {
		d.added, d.deleted = len(d.after.lines), len(d.before.lines)
		text := d.after
		if d.deleted > 0 {
			text = d.before
		}
		if len(text.lines) == 0 {
			return d, nil
		}
		d.single = &editor.State{Buffer: editor.NewBuffer(text.display), ReadOnly: true, FitContent: true,
			Font: font, LineHeight: 1.6, TabSize: 4, NoActiveLine: true, NoFolding: true, NoIndentGuides: true}
		if err := prepareDiffSyntax(ctx, path, d.single.Buffer); err != nil {
			return nil, err
		}
		return d, nil
	}
	state := &editor.DiffState{Old: editor.NewBuffer(d.before.display), New: editor.NewBuffer(d.after.display),
		Layout: editor.DiffUnified, FitContent: true, Font: font, LineHeight: 1.6, TabSize: 4,
		Indicators: editor.DiffBars, ContextLines: 3, ExpansionLines: 30}
	// Ordinary comparisons retain the native diff engine and expandable context.
	// A single blank line and an EOF-only replacement need explicit patch rows:
	// an empty Buffer otherwise means zero lines to the native comparison engine.
	switch {
	case d.before.display == "" || d.after.display == "":
		state.SetPatch(diffBlankPatch(path, d.before, d.after))
	default:
		if patch, needed := diffEOFPatch(path, d.before, d.after, state.Hunks()); needed {
			state.SetPatch(patch)
		}
	}
	for _, buffer := range []*editor.Buffer{state.Old, state.New} {
		// SetPatch owns these buffers; changing them would clear its row mapping.
		if err := prepareDiffSyntax(ctx, path, buffer); err != nil {
			return nil, err
		}
	}
	d.comparison = state
	d.added, d.deleted = state.Stats() // Warm the native comparison on the worker.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *diffDocument) View(c *ui.Context, split bool) ui.Element {
	added, deleted := diffChangeColors(c.Theme())
	theme := &editor.Theme{Added: added, Deleted: deleted, ActiveLine: ui.Transparent,
		IndentGuide: ui.Transparent, ActiveIndentGuide: ui.Transparent}
	if d.single != nil {
		color := added
		if d.deleted > 0 {
			color = deleted
		}
		theme.Background, theme.LineNumber = c.Theme().Background.Mix(color, .1), color
		d.single.Theme = theme
		return editor.View(c, d.single)
	}
	if d.comparison == nil {
		return ui.Box(c).Height(0)
	}
	d.comparison.Theme, d.comparison.Layout = theme, editor.DiffUnified
	if split {
		d.comparison.Layout = editor.DiffSplit
	}
	return editor.DiffView(c, d.comparison)
}

func diffEOFNote(before, after string) string {
	oldMissing := before != "" && !strings.HasSuffix(before, "\n")
	newMissing := after != "" && !strings.HasSuffix(after, "\n")
	switch {
	case oldMissing && newMissing:
		return "No newline at end of file (old and new)."
	case oldMissing:
		return "No newline at end of old file."
	case newMissing:
		return "No newline at end of new file."
	default:
		return ""
	}
}

// Syntax is tokenized on the retrieval worker, including the context later
// expanded by the user. The native editor only reads these immutable tokens.
func prepareDiffSyntax(ctx context.Context, path string, b *editor.Buffer) error {
	language := editor.LanguageFor(path)
	if language == nil || language.Tokenizer == nil {
		return ctx.Err()
	}
	snapshot := b.Snapshot()
	tokens := &diffTokens{lines: make([][]editor.Token, snapshot.LineCount())}
	var state uint64
	for i := range tokens.lines {
		if err := ctx.Err(); err != nil {
			return err
		}
		tokens.lines[i], state = language.Tokenizer.Tokenize(snapshot.Line(i), state, nil)
	}
	preparedLanguage := *language
	preparedLanguage.Highlighter = func(editor.Snapshot, func()) editor.Highlighter { return tokens }
	b.SetLanguage(&preparedLanguage)
	return nil
}

// Mygo compares highlighter interface identities between paints. Keep one
// comparable pointer per prepared buffer, with the immutable token slices inside.
type diffTokens struct{ lines [][]editor.Token }

func (t *diffTokens) Edit(editor.Snapshot, editor.Snapshot, []editor.Change) {}
func (t *diffTokens) Highlight(int, time.Duration) bool                      { return true }
func (t *diffTokens) Tokens(i int, _ string, dst []editor.Token) ([]editor.Token, bool) {
	if i >= 0 && i < len(t.lines) {
		dst = append(dst, t.lines[i]...)
	}
	return dst, true
}
