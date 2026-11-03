package panels

import (
	"slices"

	"github.com/egoist/mygo/ui/editor"
)

// One side is exactly one logical blank line. Preserve it as context if the
// other side has a blank line too; otherwise it is explicitly added/deleted.
func diffBlankPatch(path string, before, after diffText) editor.FilePatch {
	other, change, opposite := after.lines, "+", "-"
	if after.display == "" {
		other, change, opposite = before.lines, "-", "+"
	}
	match := -1
	for i, line := range other {
		if line == "" {
			match = i
			break
		}
	}
	var lines []string
	if match < 0 && opposite == "-" {
		lines = append(lines, "-")
	}
	for i, line := range other {
		if i == match {
			lines = append(lines, " ")
		} else {
			lines = append(lines, change+line)
		}
	}
	if match < 0 && opposite == "+" {
		lines = append(lines, "+")
	}
	return diffFilePatch(path, before, after, lines)
}

// Git considers EOF termination part of the final line. Give that metadata to
// the native line comparer using a NUL suffix (rejected in source text). These
// temporary buffers are never rendered: the patch uses original source lines.
// Keep the full-buffer view whenever the metadata does not change the hunks.
func diffEOFPatch(path string, before, after diffText, ordinary []editor.DiffHunk) (editor.FilePatch, bool) {
	if before.finalLF && after.finalLF {
		return editor.FilePatch{}, false
	}
	comparisonText := func(text diffText) string {
		if !text.finalLF {
			return text.display + "\x00"
		}
		return text.display
	}
	metadata := editor.DiffState{Old: editor.NewBuffer(comparisonText(before)), New: editor.NewBuffer(comparisonText(after))}
	hunks := metadata.Hunks()
	if slices.Equal(hunks, ordinary) {
		return editor.FilePatch{}, false
	}
	return diffHunkPatch(path, before, after, hunks), true
}

func diffHunkPatch(path string, before, after diffText, hunks []editor.DiffHunk) editor.FilePatch {
	var lines []string
	old, next := 0, 0
	for _, h := range hunks {
		for old < h.OldStart {
			lines = append(lines, " "+before.lines[old])
			old, next = old+1, next+1
		}
		for old < h.OldEnd {
			lines = append(lines, "-"+before.lines[old])
			old++
		}
		for next < h.NewEnd {
			lines = append(lines, "+"+after.lines[next])
			next++
		}
	}
	for old < len(before.lines) {
		lines = append(lines, " "+before.lines[old])
		old++
	}
	return diffFilePatch(path, before, after, lines)
}

func diffFilePatch(path string, before, after diffText, lines []string) editor.FilePatch {
	return editor.FilePatch{OldName: path, NewName: path, Hunks: []editor.PatchHunk{{
		OldStart: 1, OldLines: len(before.lines), NewStart: 1, NewLines: len(after.lines), Lines: lines,
	}}}
}
