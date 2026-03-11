package logs

import (
	"bytes"
	"io"
	"sync"
)

const defaultMaxLines = 1000

// Buffer is an io.Writer that retains the last N lines of log output.
type Buffer struct {
	mu    sync.RWMutex
	lines []string
	max   int
	cur   bytes.Buffer // incomplete line
}

// NewBuffer creates a buffer that keeps at most maxLines lines.
// If maxLines <= 0, defaultMaxLines is used.
func NewBuffer(maxLines int) *Buffer {
	if maxLines <= 0 {
		maxLines = defaultMaxLines
	}
	return &Buffer{lines: make([]string, 0, maxLines+1), max: maxLines}
}

// Write implements io.Writer. Content is split by newlines; each line is stored.
func (b *Buffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n = len(p)
	for _, c := range p {
		if c == '\n' {
			b.flushLine()
		} else {
			b.cur.WriteByte(c)
		}
	}
	return n, nil
}

func (b *Buffer) flushLine() {
	line := b.cur.String()
	b.cur.Reset()
	if line == "" {
		return
	}
	b.lines = append(b.lines, line)
	if len(b.lines) > b.max {
		b.lines = b.lines[len(b.lines)-b.max:]
	}
}

// Lines returns a copy of the buffered lines (oldest first).
func (b *Buffer) Lines() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	// Flush any partial line for display
	lines := make([]string, len(b.lines), len(b.lines)+1)
	copy(lines, b.lines)
	if b.cur.Len() > 0 {
		lines = append(lines, b.cur.String())
	}
	return lines
}

// Writer returns this buffer as an io.Writer (for use with log.SetOutput).
func (b *Buffer) Writer() io.Writer {
	return io.Writer(b)
}
