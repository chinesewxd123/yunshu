package agent

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type fileCollectorSource struct {
	logSourceID uint
	path        string // literal file path or glob pattern
	tailLines   int

	isPattern bool
	mu        sync.Mutex
	files     map[string]*trackedFileState
}

type trackedFileState struct {
	offset      int64
	initialized bool
}

// fileCollector reads multiple files in-process (no tail subprocess).
// It emits appended lines for each source with its logSourceID.
type fileCollector struct {
	mu      sync.Mutex
	sources []*fileCollectorSource
	debug   bool
}

type collectedFileLine struct {
	Line     string
	FilePath string
}

func newFileCollector(debug bool) *fileCollector {
	return &fileCollector{debug: debug}
}

func normalizeLinuxShortLogPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return p
	}
	// already absolute or contains separators: keep as-is
	if filepath.IsAbs(p) || strings.ContainsAny(p, `/\`) {
		return p
	}
	if runtime.GOOS == "windows" {
		return p
	}
	// common Linux: "messages" -> "/var/log/messages"
	return filepath.Join("/var/log", p)
}

func looksLikeDirOrGlob(p string) bool {
	p = strings.TrimSpace(p)
	if p == "" {
		return false
	}
	if strings.ContainsAny(p, "*?[]") {
		return true
	}
	// Best-effort: treat existing dir as legacy path.
	if st, err := os.Stat(p); err == nil && st.IsDir() {
		return true
	}
	return false
}

func (c *fileCollector) AddSource(logSourceID uint, path string, tailLines int) error {
	path = normalizeLinuxShortLogPath(path)
	if strings.TrimSpace(path) == "" {
		return errors.New("empty path")
	}
	if tailLines <= 0 {
		tailLines = 200
	}
	isPattern := false
	if st, err := os.Stat(path); err == nil && st.IsDir() {
		path = filepath.Join(path, "*.log")
		isPattern = true
	}
	if strings.ContainsAny(path, "*?[]") {
		isPattern = true
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sources = append(c.sources, &fileCollectorSource{
		logSourceID: logSourceID,
		path:        path,
		tailLines:   tailLines,
		isPattern:   isPattern,
		files:       map[string]*trackedFileState{},
	})
	return nil
}

func (c *fileCollector) Start(ctx context.Context, pollEvery time.Duration) <-chan collectedLine {
	if pollEvery <= 0 {
		pollEvery = 300 * time.Millisecond
	}
	out := make(chan collectedLine, 4096)

	c.mu.Lock()
	srcs := append([]*fileCollectorSource(nil), c.sources...)
	c.mu.Unlock()

	// One watcher for all sources; if it fails, we still poll.
	w, _ := fsnotify.NewWatcher()
	if w != nil {
		// Close watcher on exit.
		go func() {
			<-ctx.Done()
			_ = w.Close()
		}()
	}

	// map watched dir -> count
	dirRef := map[string]int{}
	addWatchDir := func(dir string) {
		if w == nil || dir == "" {
			return
		}
		if dirRef[dir] == 0 {
			_ = w.Add(dir)
		}
		dirRef[dir]++
	}
	for _, s := range srcs {
		addWatchDir(filepath.Dir(s.path))
	}

	notify := make(chan struct{}, 1)
	if w != nil {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case ev, ok := <-w.Events:
					if !ok {
						return
					}
					// Any relevant event: trigger a poll sooner.
					if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) != 0 {
						select {
						case notify <- struct{}{}:
						default:
						}
					}
				case <-w.Errors:
					// ignore; polling still works
				}
			}
		}()
	}

	go func() {
		defer close(out)

		ticker := time.NewTicker(pollEvery)
		defer ticker.Stop()

		pollAll := func() {
			for _, s := range srcs {
				lines, err := collectAppendedLinesBySource(s)
				if err != nil {
					if c.debug {
						println("[agent][file] collect error source=", s.logSourceID, " path=", s.path, " err=", err.Error())
					}
					continue
				}
				if c.debug && len(lines) > 0 {
					println("[agent][file] collected lines source=", s.logSourceID, " count=", len(lines))
				}
				for _, ln := range lines {
					if ln.Line == "" {
						continue
					}
					out <- collectedLine{logSourceID: s.logSourceID, line: ln.Line, filePath: ln.FilePath}
				}
			}
		}

		// First poll soon after start to catch fast writers.
		pollAll()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pollAll()
			case <-notify:
				pollAll()
			}
		}
	}()

	return out
}

func initialOffsetByTailLines(path string, tailLines int) (int64, error) {
	if tailLines <= 0 {
		return 0, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return 0, err
	}
	size := st.Size()
	if size <= 0 {
		return 0, nil
	}

	// Read backwards in chunks until we have N newlines.
	const chunkSize = 64 * 1024
	var (
		buf      []byte
		readSize int64
		newlines int
		pos      = size
	)
	for pos > 0 && newlines <= tailLines {
		n := int64(chunkSize)
		if pos-n < 0 {
			n = pos
		}
		pos -= n
		tmp := make([]byte, n)
		if _, err := f.ReadAt(tmp, pos); err != nil && !errors.Is(err, io.EOF) {
			return 0, err
		}
		readSize += n
		buf = append(tmp, buf...)
		newlines = bytes.Count(buf, []byte{'\n'})
		// cap buffer growth
		if readSize > 8*1024*1024 {
			break
		}
	}
	if len(buf) == 0 {
		return 0, nil
	}
	// Find start offset of last N lines
	if tailLines <= 0 {
		return 0, nil
	}
	need := tailLines
	idx := len(buf)
	for idx > 0 && need > 0 {
		idx = bytes.LastIndexByte(buf[:idx], '\n')
		if idx < 0 {
			return 0, nil
		}
		need--
	}
	// idx is position of the newline before the last N lines
	startInBuf := 0
	if idx >= 0 {
		startInBuf = idx + 1
	}
	startOffset := size - int64(len(buf)) + int64(startInBuf)
	if startOffset < 0 {
		startOffset = 0
	}
	return startOffset, nil
}

func collectAppendedLinesBySource(s *fileCollectorSource) ([]collectedFileLine, error) {
	targets := make([]string, 0, 8)
	if s.isPattern {
		matched, err := filepath.Glob(s.path)
		if err != nil {
			return nil, err
		}
		targets = append(targets, matched...)
	} else {
		targets = append(targets, s.path)
	}
	if len(targets) == 0 {
		return nil, nil
	}
	sort.Strings(targets)

	out := make([]collectedFileLine, 0, 256)
	for _, p := range targets {
		lines, err := readAppendedLinesFromPath(s, p)
		if err != nil {
			continue
		}
		out = append(out, lines...)
	}
	return out, nil
}

func readAppendedLinesFromPath(s *fileCollectorSource, path string) ([]collectedFileLine, error) {
	s.mu.Lock()
	st, ok := s.files[path]
	if !ok {
		st = &trackedFileState{}
		s.files[path] = st
	}
	s.mu.Unlock()

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := info.Size()
	if !st.initialized {
		off, err := initialOffsetByTailLines(path, s.tailLines)
		if err == nil {
			st.offset = off
			st.initialized = true
		}
	}
	offset := st.offset
	if size < offset {
		// truncated/rotated
		offset = 0
	}
	if size == offset {
		return nil, nil
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}

	sc := bufio.NewScanner(f)
	// allow longer lines
	sc.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	lines := make([]collectedFileLine, 0, 256)
	var consumed int64
	for sc.Scan() {
		txt := sc.Text()
		lines = append(lines, collectedFileLine{
			Line:     txt,
			FilePath: path,
		})
		consumed += int64(len(txt)) + 1 // + '\n' best-effort
		// soft cap per poll to avoid huge bursts blocking loop
		if len(lines) >= 2000 {
			break
		}
	}
	// Even if scanner error (e.g., token too long), move offset by bytes read so far.
	newOffset := offset + consumed
	// Clamp to file size.
	if newOffset > size {
		newOffset = size
	}

	st.offset = newOffset
	st.initialized = true

	return lines, nil
}
