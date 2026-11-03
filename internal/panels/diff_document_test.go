package panels

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

func TestDiffStatusKeepsIndexAndWorkingChangesDistinct(t *testing.T) {
	text := "1 MM N... 100644 100644 100644 aaaa bbbb src/main.go\x00" +
		"2 R. N... 100644 100644 100644 cccc cccc R100 new name.go\x00old\tname.go\x00" +
		"? new\nfile.txt\x00"
	entries, err := parseDiffStatus(text)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d entries", len(entries))
	}
	versions := entries[0].sources()
	if len(versions) != 2 {
		t.Fatalf("got %d changes for a staged and unstaged file", len(versions))
	}
	if versions[0].scope != "Staged" || versions[0].before.object != "aaaa" || versions[0].after.object != "bbbb" {
		t.Fatalf("staged change must compare HEAD to index: %+v", versions[0])
	}
	if versions[1].scope != "Unstaged" || versions[1].before.object != "bbbb" || !versions[1].after.worktree {
		t.Fatalf("unstaged change must compare index to disk: %+v", versions[1])
	}
	rename := entries[1].sources()[0]
	if rename.path != "new name.go" || rename.oldPath != "old\tname.go" || rename.status != "Renamed" {
		t.Fatalf("rename paths lost their identity: %+v", rename)
	}
	if entries[2].path != "new\nfile.txt" || entries[2].sources()[0].scope != "Untracked" {
		t.Fatalf("untracked path: %+v", entries[2])
	}
}

func TestDiffUnbornAndDeletedVersionsAreEmpty(t *testing.T) {
	entries, err := parseDiffStatus("1 A. N... 000000 100644 100644 " + strings.Repeat("0", 40) + " abc123 first.go\x00" +
		"1 .D N... 100644 100644 000000 def456 def456 deleted.go\x00")
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []diffVersion{entries[0].sources()[0].before, entries[1].sources()[0].after} {
		text, err := readDiffVersion(context.Background(), "", "", version, 1024, nil)
		if err != nil || text != "" {
			t.Fatalf("missing version must be empty: %q, %v", text, err)
		}
	}
	_, err = parseDiffStatus("2 R. N... 100644 100644 100644 aaa bbb R100 new.go\x00")
	if err == nil {
		t.Fatal("an incomplete rename must not invent the old path")
	}
}

func TestDiffPreviewBudgetSurvivesRejectedVersions(t *testing.T) {
	contents := map[string]string{"old": "old\n", "new": "new content\n", "image": "\x00PNG", "invalid": "\xff"}
	read := func(version diffVersion, limit int) (string, error) {
		text, ok := contents[version.object]
		if !ok {
			return "", errors.New("missing object")
		}
		return decodeDiffText([]byte(text), limit)
	}
	for _, tc := range []struct{ name, before, after string }{
		{"binary before", "image", "new"},
		{"binary after", "old", "image"},
		{"non-UTF-8 after", "old", "invalid"},
		{"failed after", "old", "missing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			budget := len(contents["old"]) + len(contents["new"])
			remaining := budget
			source := diffSource{path: "file.txt", scope: "Staged", before: diffVersion{object: tc.before}, after: diffVersion{object: tc.after}}
			for range 4 {
				file := loadDiffFile(context.Background(), source, &remaining, read)
				if file.diff != nil || file.note == "" || remaining != budget {
					t.Fatalf("rejected preview spent text budget: remaining=%d, note=%q", remaining, file.note)
				}
			}
			source.before.object, source.after.object = "old", "new"
			file := loadDiffFile(context.Background(), source, &remaining, read)
			if file.diff == nil || remaining != 0 || file.added != 1 || file.deleted != 1 {
				t.Fatalf("valid text must still fit: remaining=%d, +%d -%d, note=%q", remaining, file.added, file.deleted, file.note)
			}
		})
	}
}

type diffTestPipe struct{ io.Writer }

func (diffTestPipe) Close() error { return nil }

func TestDiffBatchPreflightsSizeAndKeepsResponseFraming(t *testing.T) {
	var requests bytes.Buffer
	batch := &diffBatch{ctx: context.Background(), cancel: func() {}, stdin: diffTestPipe{&requests},
		reader: bufio.NewReader(strings.NewReader("large blob 314572800\nmissing missing\ntree tree 12\nvalid blob 4\nvalid blob 4\ntext\n"))}
	if _, err := batch.readBlob("large", diffFileLimit); !errors.Is(err, errDiffLimit) {
		t.Fatalf("oversized metadata: %v", err)
	}
	if _, err := batch.readBlob("missing", diffFileLimit); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("missing object: %v", err)
	}
	if _, err := batch.readBlob("tree", diffFileLimit); err == nil || !strings.Contains(err.Error(), "tree") {
		t.Fatalf("non-blob object: %v", err)
	}
	data, err := batch.readBlob("valid", 4)
	if err != nil || string(data) != "text" {
		t.Fatalf("subsequent text object: %q, %v", data, err)
	}
	if want := "info large\ninfo missing\ninfo tree\ninfo valid\ncontents valid\n"; requests.String() != want {
		t.Fatalf("only size-eligible blobs should request contents: %q", requests.String())
	}
}

func TestDiffCountsSourceLinesWithoutFinalCaretLine(t *testing.T) {
	for _, tc := range []struct {
		name, before, after string
		added, deleted      int
		rows                int
		note                string
	}{
		{"new file", "", "package main\n", 1, 0, 1, ""},
		{"deleted file", "one\n", "", 0, 1, 1, ""},
		{"replacement", "before\n", "after\n", 1, 1, 2, ""},
		{"empty file", "", "", 0, 0, 0, ""},
		{"added blank line", "", "\n", 1, 0, 1, ""},
		{"deleted blank line", "\n", "", 0, 1, 1, ""},
		{"trailing blank line", "one\n", "one\n\n", 1, 0, 2, ""},
		{"blank to text", "\n", "first\n\nlast\n", 2, 0, 3, ""},
		{"text to blank", "first\n\nlast\n", "\n", 0, 2, 3, ""},
		{"added final newline", "same", "same\n", 1, 1, 2, "No newline at end of old file."},
		{"removed final newline", "same\n", "same", 1, 1, 2, "No newline at end of new file."},
		{"missing final newline", "before", "after", 1, 1, 2, "No newline at end of file (old and new)."},
		{"unterminated line becomes context", "one", "one\nsecond", 2, 1, 3, "No newline at end of file (old and new)."},
		{"added unterminated line", "one\n", "one\nsecond", 1, 0, 2, "No newline at end of new file."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := newDiffDocument(context.Background(), "main.go", tc.before, tc.after)
			if err != nil {
				t.Fatal(err)
			}
			if d.added != tc.added || d.deleted != tc.deleted {
				t.Fatalf("got +%d -%d; want +%d -%d", d.added, d.deleted, tc.added, tc.deleted)
			}
			if d.before.raw != tc.before || d.after.raw != tc.after || diffEOFNote(tc.before, tc.after) != tc.note {
				t.Fatal("original source and EOF metadata must survive display normalization")
			}
			if d.comparison != nil && (d.comparison.Old.String() != d.before.display || d.comparison.New.String() != d.after.display) {
				t.Fatal("native patch buffers must contain the actual source lines")
			}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Scroll(c).Fill().Children(func() {
					d.View(c.Key("changes-file-diff"), false).Label("Source diff").FillWidth().Shrink(0)
				})
			}, 600, 400)
			tt.Frame()
			bounds, ok := tt.Find("Source diff")
			if !ok {
				t.Fatal("diff view was not laid out")
			}
			positions, rowHeight := renderedDiffRows(d)
			if len(positions) != tc.rows {
				t.Fatalf("rendered source mapping has %d rows, want %d: %v", len(positions), tc.rows, positions)
			}
			if tc.rows == 0 {
				if bounds.H != 0 {
					t.Fatalf("empty source has a rendered row: %+v", bounds)
				}
				return
			}
			lastBottom := positions[len(positions)-1] + rowHeight
			if rowHeight < 12 || rowHeight > 32 || math.Abs(float64(bounds.H)-lastBottom) > .5 {
				t.Fatalf("view extends beyond its real source rows: height=%.1f, last row bottom=%.1f, row height=%.1f", bounds.H, lastBottom, rowHeight)
			}
		})
	}
}

// Measure the native widgets' displayed line positions, merging the two sides'
// shared context. Comparing the final row's bottom with actual view bounds also
// detects a painted caret row that is absent from the source mapping.
func renderedDiffRows(d *diffDocument) ([]float64, float64) {
	var positions []float64
	var rowHeight float64
	if d.single != nil {
		for i := range max(len(d.before.lines), len(d.after.lines)) {
			positions = append(positions, d.single.LineTop(i))
		}
		rowHeight = d.single.LineTop(1) - d.single.LineTop(0)
	} else if d.comparison != nil {
		for i := range d.before.lines {
			positions = append(positions, d.comparison.LineTop(editor.DiffOld, i))
		}
		for i := range d.after.lines {
			positions = append(positions, d.comparison.LineTop(editor.DiffNew, i))
		}
		slices.Sort(positions)
		positions = slices.Compact(positions)
		rowHeight = d.comparison.LineTop(editor.DiffNew, 1) - d.comparison.LineTop(editor.DiffNew, 0)
		for i := 1; i < len(positions); i++ {
			step := positions[i] - positions[i-1]
			if rowHeight <= 0 || step < rowHeight {
				rowHeight = step
			}
		}
	}
	return positions, rowHeight
}

func TestHighlightedDiffPaintsRepeatedFrames(t *testing.T) {
	doc, err := newDiffDocument(context.Background(), "greeting.go",
		"package greeting\n\nfunc Message() string { return \"before\" }\n",
		"package greeting\n\nfunc Message() string { return \"after\" }\n")
	if err != nil {
		t.Fatal(err)
	}
	// Split paints both prepared highlighters, including the old side. Repaint
	// the same document so mygo follows their identities across actual frames.
	painted := 0
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().DrawOver(func(*ui.Painter, ui.Rect) { painted++ }).Children(func() {
			ui.Scroll(c).Fill().Children(func() {
				doc.View(c.Key("changes-file-diff"), true).Label("Staged diff — greeting.go").FillWidth().Shrink(0)
			})
		})
	}, 800, 400)
	first := painted
	tt.Frame() // Tester.Frame explicitly requests and paints a fresh frame.
	if first == 0 || painted <= first {
		t.Fatalf("expected repeated painted frames, got %d then %d", first, painted)
	}
	for _, label := range []string{"Old text", "New text"} {
		bounds, ok := tt.Find(label)
		if !ok || bounds.W <= 0 || bounds.H <= 0 {
			t.Fatalf("highlighted side %q was not visible", label)
		}
	}
	bounds, ok := tt.Find("Staged diff — greeting.go")
	if !ok || bounds.H < 40 || bounds.H > 160 || bounds.Y+bounds.H > 400 {
		t.Fatalf("short diff should fit comfortably in its viewport: %+v, found=%v", bounds, ok)
	}
	rowHeight := doc.comparison.LineTop(editor.DiffNew, 1) - doc.comparison.LineTop(editor.DiffNew, 0)
	if rowHeight < 12 || rowHeight > 32 {
		t.Fatalf("code rows should have readable, compact spacing; got %.1f points", rowHeight)
	}
}
