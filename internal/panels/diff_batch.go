package panels

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

// One worker owns one batch process for the refresh. Requests are synchronous:
// info checks type/size before contents, so an oversized blob is never streamed.
type diffBatch struct {
	ctx    context.Context
	cancel context.CancelFunc
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	reader *bufio.Reader
	stderr cappedOutput
	err    error
}

func newDiffBatch(ctx context.Context, root string) (*diffBatch, error) {
	processContext, cancel := context.WithCancel(ctx)
	cmd := diffGitCommand(processContext, root, "cat-file", "--batch-command")
	b := &diffBatch{ctx: ctx, cancel: cancel, cmd: cmd, stderr: cappedOutput{limit: 4096}}
	cmd.Stderr = &b.stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		cmd.Stdin.(io.Closer).Close() // StdinPipe's child end; Start was not called.
		cancel()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		cancel() // Start closes its pipes on failure.
		return nil, err
	}
	b.stdin, b.reader = stdin, bufio.NewReader(stdout)
	return b, nil
}

func (b *diffBatch) Close() error {
	// EOF ends cat-file normally. CommandContext also kills it on deadline or
	// panel close; Wait reaps it and its pipe copier before stderr is read.
	closeErr := b.stdin.Close()
	waitErr := b.cmd.Wait()
	b.cancel()
	if b.ctx.Err() != nil {
		return b.ctx.Err()
	}
	if b.err != nil {
		if detail := strings.TrimSpace(b.stderr.String()); detail != "" {
			return fmt.Errorf("%w: %s", b.err, detail)
		}
		return b.err
	}
	if waitErr != nil {
		return fmt.Errorf("Git object reader: %w: %s", waitErr, strings.TrimSpace(b.stderr.String()))
	}
	return closeErr
}

type diffObjectInfo struct {
	kind string
	size int64
}

func parseDiffObjectInfo(header string) (diffObjectInfo, error) {
	parts := strings.Fields(header)
	if len(parts) == 2 {
		return diffObjectInfo{}, fmt.Errorf("Git object %s: %s", parts[0], parts[1])
	}
	if len(parts) != 3 {
		return diffObjectInfo{}, errors.New("Invalid Git object header")
	}
	size, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || size < 0 {
		return diffObjectInfo{}, errors.New("Invalid Git object size")
	}
	return diffObjectInfo{kind: parts[1], size: size}, nil
}

func (b *diffBatch) request(command, object string) (diffObjectInfo, error) {
	if b.err != nil {
		return diffObjectInfo{}, b.err
	}
	if err := b.ctx.Err(); err != nil {
		return diffObjectInfo{}, err
	}
	if _, err := io.WriteString(b.stdin, command+" "+object+"\n"); err != nil {
		return diffObjectInfo{}, b.abort(err)
	}
	// ReadSlice bounds the protocol header to the reader's 4 KiB buffer.
	header, err := b.reader.ReadSlice('\n')
	if err != nil {
		return diffObjectInfo{}, b.abort(err)
	}
	info, err := parseDiffObjectInfo(string(header))
	if err != nil && len(strings.Fields(string(header))) != 2 {
		return diffObjectInfo{}, b.abort(err)
	}
	return info, err
}

func (b *diffBatch) readBlob(object string, limit int) ([]byte, error) {
	info, err := b.request("info", object)
	if err != nil {
		return nil, err
	}
	if info.kind != "blob" {
		return nil, fmt.Errorf("Git object %s is %s, not a file", object, info.kind)
	}
	if info.size > int64(limit) {
		return nil, errDiffLimit
	}
	contents, err := b.request("contents", object)
	if err != nil {
		return nil, b.abort(err)
	}
	if contents != info {
		return nil, b.abort(errors.New("Git object metadata changed while reading"))
	}
	data := make([]byte, int(info.size)+1)
	if _, err := io.ReadFull(b.reader, data); err != nil {
		return nil, b.abort(err)
	}
	if data[len(data)-1] != '\n' {
		return nil, b.abort(errors.New("Invalid Git object terminator"))
	}
	return data[:len(data)-1], nil
}

func (b *diffBatch) abort(err error) error {
	if b.err != nil {
		return b.err
	}
	if b.ctx.Err() != nil {
		err = b.ctx.Err()
	}
	b.err = fmt.Errorf("Git object stream: %w", err)
	b.cancel()
	return b.err
}
