package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Item struct {
	FileID  int64
	RelPath string
	Size    int64
}

type EventKind int

const (
	Started EventKind = iota
	Progress
	Completed
	Failed
)

type Event struct {
	Kind    EventKind
	FileID  int64
	RelPath string
	Done    int64
	Total   int64
	Speed   float64 // bytes/sec since last progress event
	Err     error
}

// URLResolver turns a put.io file ID into a download URL. Injected so the
// engine is testable without put.io.
type URLResolver func(ctx context.Context, fileID int64) (string, error)

type Manager struct {
	resolve URLResolver
	destDir string
	httpc   *http.Client
	events  chan Event

	mu      sync.Mutex
	queue   []Item
	active  bool
	cancels map[int64]context.CancelFunc
}

func New(resolve URLResolver, destDir string) *Manager {
	return &Manager{
		resolve: resolve,
		destDir: destDir,
		httpc:   &http.Client{},
		events:  make(chan Event, 64),
		cancels: make(map[int64]context.CancelFunc),
	}
}

func (m *Manager) Events() <-chan Event { return m.events }

func (m *Manager) Enqueue(items []Item) {
	m.mu.Lock()
	m.queue = append(m.queue, items...)
	start := !m.active
	if start {
		m.active = true
	}
	m.mu.Unlock()
	if start {
		go m.run()
	}
}

// Cancel drops a queued item or aborts it mid-download.
func (m *Manager) Cancel(fileID int64) {
	m.mu.Lock()
	for i, it := range m.queue {
		if it.FileID == fileID {
			m.queue = append(m.queue[:i], m.queue[i+1:]...)
			m.mu.Unlock()
			m.events <- Event{Kind: Failed, FileID: it.FileID, RelPath: it.RelPath, Err: context.Canceled}
			return
		}
	}
	cancel, ok := m.cancels[fileID]
	m.mu.Unlock()
	if ok {
		cancel()
	}
}

func (m *Manager) run() {
	for {
		m.mu.Lock()
		if len(m.queue) == 0 {
			m.active = false
			m.mu.Unlock()
			return
		}
		it := m.queue[0]
		m.queue = m.queue[1:]
		ctx, cancel := context.WithCancel(context.Background())
		m.cancels[it.FileID] = cancel
		m.mu.Unlock()

		err := m.download(ctx, it)

		m.mu.Lock()
		delete(m.cancels, it.FileID)
		m.mu.Unlock()
		cancel()

		if err != nil {
			m.events <- Event{Kind: Failed, FileID: it.FileID, RelPath: it.RelPath, Err: err}
		} else {
			m.events <- Event{Kind: Completed, FileID: it.FileID, RelPath: it.RelPath, Done: it.Size, Total: it.Size}
		}
	}
}

func (m *Manager) download(ctx context.Context, it Item) error {
	url, err := m.resolve(ctx, it.FileID)
	if err != nil {
		return fmt.Errorf("resolving url: %w", err)
	}

	final := filepath.Join(m.destDir, it.RelPath)
	part := final + ".part"
	if err := os.MkdirAll(filepath.Dir(final), 0o755); err != nil {
		return err
	}

	var offset int64
	if fi, err := os.Stat(part); err == nil {
		offset = fi.Size()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}

	resp, err := m.httpc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	flags := os.O_CREATE | os.O_WRONLY
	switch resp.StatusCode {
	case http.StatusPartialContent:
		flags |= os.O_APPEND
	case http.StatusOK:
		offset = 0 // server ignored the range; start over
		flags |= os.O_TRUNC
	default:
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	f, err := os.OpenFile(part, flags, 0o644)
	if err != nil {
		return err
	}

	total := it.Size
	m.events <- Event{Kind: Started, FileID: it.FileID, RelPath: it.RelPath, Done: offset, Total: total}

	if err := m.copyWithProgress(ctx, f, resp.Body, it, offset, total); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(part, final)
}

func (m *Manager) copyWithProgress(ctx context.Context, dst io.Writer, src io.Reader, it Item, offset, total int64) error {
	buf := make([]byte, 256*1024)
	done := offset
	lastEmit := time.Now()
	lastDone := done
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, rerr := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return werr
			}
			done += int64(n)
		}
		if since := time.Since(lastEmit); since >= 250*time.Millisecond {
			ev := Event{
				Kind: Progress, FileID: it.FileID, RelPath: it.RelPath,
				Done: done, Total: total,
				Speed: float64(done-lastDone) / since.Seconds(),
			}
			select { // progress is best-effort; never stall the transfer
			case m.events <- ev:
			default:
			}
			lastEmit = time.Now()
			lastDone = done
		}
		if rerr == io.EOF {
			return nil
		}
		if rerr != nil {
			return rerr
		}
	}
}
