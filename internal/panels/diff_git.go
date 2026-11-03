package panels

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	diffFileLimit  = 512 * 1024
	diffTotalLimit = 2 * 1024 * 1024
	diffFileCount  = 200
)

type cappedOutput struct {
	bytes.Buffer
	limit     int
	truncated bool
}

func (b *cappedOutput) Write(data []byte) (int, error) {
	n := len(data)
	remaining := max(0, b.limit-b.Len())
	b.truncated = b.truncated || n > remaining
	b.Buffer.Write(data[:min(n, remaining)])
	return n, nil
}

func gitOutput(ctx context.Context, root string, output *cappedOutput, args ...string) error {
	cmd := diffGitCommand(ctx, root, args...)
	stderr := cappedOutput{limit: 4096}
	cmd.Stdout, cmd.Stderr = output, &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func repositoryRoot(ctx context.Context, directory string) (string, error) {
	output := cappedOutput{limit: 8192}
	if err := gitOutput(ctx, directory, &output, "rev-parse", "--show-toplevel"); err != nil {
		return "", err
	}
	root := strings.TrimSuffix(output.String(), "\n")
	if output.truncated || root == "" {
		return "", fmt.Errorf("Git returned an unavailable repository root")
	}
	return root, nil
}

// Porcelain v2 supplies the exact HEAD/index objects even in an unborn
// repository. NUL records preserve spaces, tabs and newlines in paths.
type diffEntry struct {
	path, oldPath, xy             string
	headMode, indexMode, workMode string
	head, index                   string
	conflict                      bool
}

func parseDiffStatus(text string) ([]diffEntry, error) {
	var entries []diffEntry
	records := strings.Split(text, "\x00")
	for i := 0; i < len(records); i++ {
		record := records[i]
		if record == "" || record[0] == '#' || record[0] == '!' {
			continue
		}
		if strings.HasPrefix(record, "? ") {
			entries = append(entries, diffEntry{path: record[2:], xy: "??"})
			continue
		}
		fields := 9
		switch record[0] {
		case '2':
			fields = 10
		case 'u':
			fields = 11
		case '1':
		default:
			return entries, fmt.Errorf("unrecognized Git status record")
		}
		parts := strings.SplitN(record, " ", fields)
		if len(parts) != fields || len(parts[1]) != 2 || parts[fields-1] == "" {
			return entries, fmt.Errorf("incomplete Git status record")
		}
		e := diffEntry{path: parts[fields-1], xy: parts[1], conflict: record[0] == 'u'}
		if !e.conflict {
			e.headMode, e.indexMode, e.workMode = parts[3], parts[4], parts[5]
			e.head, e.index = parts[6], parts[7]
		}
		if record[0] == '2' {
			i++
			if i >= len(records) || records[i] == "" {
				return entries, fmt.Errorf("incomplete Git rename record")
			}
			e.oldPath = records[i]
		}
		entries = append(entries, e)
	}
	return entries, nil
}

type diffVersion struct {
	object, mode string
	worktree     bool
}

type diffSource struct {
	path, oldPath, scope, status string
	before, after                diffVersion
	conflict                     bool
}

func (e diffEntry) sources() []diffSource {
	if e.conflict {
		return []diffSource{{path: e.path, scope: "Unmerged", status: "Conflict", conflict: true}}
	}
	if e.xy == "??" {
		return []diffSource{{path: e.path, scope: "Untracked", status: "New file", after: diffVersion{worktree: true}}}
	}
	var out []diffSource
	if e.xy[0] != '.' {
		oldPath := ""
		if e.xy[0] == 'R' || e.xy[0] == 'C' {
			oldPath = e.oldPath
		}
		out = append(out, diffSource{path: e.path, oldPath: oldPath, scope: "Staged", status: diffStatusLabel(e.xy[0]),
			before: diffVersion{object: e.head, mode: e.headMode}, after: diffVersion{object: e.index, mode: e.indexMode}})
	}
	if e.xy[1] != '.' {
		oldPath := ""
		if e.xy[1] == 'R' || e.xy[1] == 'C' {
			oldPath = e.oldPath
		}
		out = append(out, diffSource{path: e.path, oldPath: oldPath, scope: "Unstaged", status: diffStatusLabel(e.xy[1]),
			before: diffVersion{object: e.index, mode: e.indexMode}, after: diffVersion{mode: e.workMode, worktree: true}})
	}
	return out
}

func diffStatusLabel(status byte) string {
	switch status {
	case 'A':
		return "Added"
	case 'D':
		return "Deleted"
	case 'R':
		return "Renamed"
	case 'C':
		return "Copied"
	case 'T':
		return "Type changed"
	default:
		return "Modified"
	}
}

type diffSnapshot struct {
	root      string
	files     []*diffFile
	fileCount int
	added     int
	deleted   int
	notice    string
}

func loadDiffSnapshot(ctx context.Context, directory string) (result diffSnapshot, err error) {
	root, err := repositoryRoot(ctx, directory)
	result.root = root
	if err != nil {
		return result, err
	}
	output := cappedOutput{limit: diffFileLimit}
	if err := gitOutput(ctx, root, &output, "status", "--porcelain=v2", "-z", "--untracked-files=all", "--ignore-submodules=none"); err != nil {
		return result, err
	}
	text := output.String()
	if output.truncated {
		text = text[:strings.LastIndexByte(text, 0)+1]
		result.notice = "The file list is truncated. Refresh after narrowing the working tree."
	}
	entries, err := parseDiffStatus(text)
	if err != nil && !output.truncated {
		return result, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })
	result.fileCount = len(entries)
	if len(entries) > diffFileCount {
		result.notice = strings.TrimSpace(result.notice + fmt.Sprintf(" Showing the first %d of %d changed files.", diffFileCount, len(entries)))
		entries = entries[:diffFileCount]
	}
	if len(entries) == 0 {
		return result, nil
	}
	batch, err := newDiffBatch(ctx, root)
	if err != nil {
		return result, err
	}
	defer func() {
		if closeErr := batch.Close(); closeErr != nil && ctx.Err() == nil {
			result.notice = strings.TrimSpace(result.notice + " " + closeErr.Error())
		}
	}()
	remaining := diffTotalLimit
	limited := false
	for _, entry := range entries {
		for _, source := range entry.sources() {
			file := loadDiffFile(ctx, source, &remaining, func(version diffVersion, limit int) (string, error) {
				return readDiffVersion(ctx, root, source.path, version, limit, batch)
			})
			result.files = append(result.files, file)
			result.added += file.added
			result.deleted += file.deleted
			limited = limited || file.limited
		}
	}
	if limited {
		result.notice = strings.TrimSpace(result.notice + " Some previews are limited to 512 KiB per file and 2 MiB in total. Counts cover the displayed text.")
	}
	if ctx.Err() != nil {
		result.notice = strings.TrimSpace(result.notice + " Reading changes took too long. Some previews are unavailable; refresh to try again.")
	}
	return result, nil
}

var errDiffLimit = errors.New("Preview exceeds the 512 KiB file limit or the remaining 2 MiB preview budget")

func readDiffVersion(ctx context.Context, root, path string, version diffVersion, limit int, batch *diffBatch) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if version.mode == "000000" || (!version.worktree && strings.Trim(version.object, "0") == "") {
		return "", nil
	}
	var data []byte
	var err error
	if version.worktree {
		data, err = readDiffWorktree(filepath.Join(root, path), limit)
	} else {
		data, err = batch.readBlob(version.object, limit)
	}
	if err != nil {
		return "", err
	}
	return decodeDiffText(data, limit)
}

func readDiffWorktree(path string, limit int) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		return []byte(target), err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("This path is not a regular text file")
	}
	if info.Size() > int64(limit) {
		return nil, errDiffLimit
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, int64(limit)+1))
}

func decodeDiffText(data []byte, limit int) (string, error) {
	if len(data) > limit {
		return "", errDiffLimit
	}
	if bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data) {
		return "", errDiffBinary
	}
	return string(data), nil
}

var errDiffBinary = errors.New("Binary or non-UTF-8 file — text preview unavailable")
