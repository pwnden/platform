package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/parser"
	"github.com/charmbracelet/x/vt"
)

// The server consumes output regardless of browser attachment. A slow browser
// loses its attachment, while the shell continues into the bounded VT screen.
type terminalProcess struct {
	mu         sync.Mutex
	shell      TerminalSession
	screen     *vt.Emulator
	boundary   *ansi.Parser
	pending    []byte
	unsafe     bool
	answering  atomic.Bool
	inputs     chan []byte
	modes      map[ansi.Mode]bool
	main       string
	attachment *TerminalAttachment
	done       chan struct{}
	ended      bool
	seen       bool
	code       int
	err        error
	once       sync.Once
	closeErr   error
}

func newTerminalProcess(shell TerminalSession, cols, rows int) *terminalProcess {
	p := &terminalProcess{shell: shell, screen: vt.NewEmulator(cols, rows), boundary: ansi.NewParser(), modes: map[ansi.Mode]bool{}, done: make(chan struct{}), inputs: make(chan []byte, 16)}
	p.answering.Store(true)
	p.boundary.SetDataSize(16 << 10)
	p.screen.SetScrollbackSize(500)
	p.screen.SetCallbacks(vt.Callbacks{
		EnableMode:  func(mode ansi.Mode) { p.modes[mode] = true },
		DisableMode: func(mode ansi.Mode) { p.modes[mode] = false },
	})
	p.screen.RegisterCsiHandler(int('h')|int('?')<<8, func(params ansi.Params) bool {
		for _, param := range params {
			n := param.Param(0)
			if (n == 1049 || n == 1047 || n == 47) && !p.screen.IsAltScreen() {
				p.main = p.primary()
			}
		}
		return false
	})
	go p.replies()
	go func() {
		for {
			select {
			case <-p.done:
				return
			case input := <-p.inputs:
				if _, err := p.shell.Write(input); err != nil {
					return
				}
			}
		}
	}()
	go p.read()
	return p
}
func (p *terminalProcess) render() string {
	position := p.screen.CursorPosition()
	return "\x1b[H\x1b[2J" + strings.ReplaceAll(p.screen.Render(), "\n", "\r\n") + fmt.Sprintf("\x1b[%d;%dH", position.Y+1, position.X+1)
}
func (p *terminalProcess) snapshot() []byte {
	var b strings.Builder
	b.WriteString("\x1bc")
	if p.screen.IsAltScreen() {
		b.WriteString(p.main)
		b.WriteString("\x1b[?1049h")
	}
	if p.screen.IsAltScreen() {
		b.WriteString(p.render())
	} else {
		b.WriteString(p.primary())
	}
	for mode, set := range p.modes {
		if mode == ansi.DECMode(1049) || mode == ansi.DECMode(1047) || mode == ansi.DECMode(47) {
			continue
		}
		if set {
			b.WriteString(ansi.SetMode(mode))
		} else {
			b.WriteString(ansi.ResetMode(mode))
		}
	}
	b.Write(p.pending)
	return []byte(b.String())
}

// While detached, the mirror answers queries so programs still have a terminal.
// While attached, xterm supplies the answers and the mirror remains silent.
func (p *terminalProcess) replies() {
	buffer := make([]byte, 4096)
	for {
		n, err := p.screen.Read(buffer)
		if n > 0 {
			if p.answering.Load() {
				select {
				case p.inputs <- append([]byte(nil), buffer[:n]...):
				default:
				}
			}
		}
		if err != nil {
			return
		}
	}
}
func (p *terminalProcess) primary() string {
	var b strings.Builder
	for _, line := range p.screen.Scrollback().Lines() {
		b.WriteString(line.Render())
		b.WriteString("\r\n")
	}
	if p.screen.ScrollbackLen() > 0 {
		b.WriteString(strings.Repeat("\r\n", p.screen.Height()))
	}
	b.WriteString(p.render())
	return b.String()
}
func (p *terminalProcess) read() {
	defer close(p.done)
	buffer := make([]byte, 16<<10)
	for {
		n, err := p.shell.Read(buffer)
		p.mu.Lock()
		if n > 0 {
			p.seen = true
			for _, b := range buffer[:n] {
				p.boundary.Advance(b)
				if p.boundary.State() == parser.GroundState {
					p.pending = p.pending[:0]
					p.unsafe = false
				} else if !p.unsafe {
					if len(p.pending) == 16<<10 {
						p.pending = p.pending[:0]
						p.unsafe = true
					} else {
						p.pending = append(p.pending, b)
					}
				}
			}
			p.screen.Write(buffer[:n])
			if a := p.attachment; a != nil {
				select {
				case a.frames <- append([]byte(nil), buffer[:n]...):
				default:
					close(a.notify)
					p.attachment = nil
					p.answering.Store(true)
				}
			}
		}
		p.mu.Unlock()
		if err != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			code, waitErr := p.shell.Wait(ctx)
			cancel()
			failure := p.shell.Close()
			p.mu.Lock()
			p.code = code
			p.err = errors.Join(waitErr, failure)
			p.ended = true
			if a := p.attachment; a != nil {
				close(a.notify)
				p.attachment = nil
			}
			p.mu.Unlock()
			return
		}
	}
}
func (p *terminalProcess) finished() bool { p.mu.Lock(); defer p.mu.Unlock(); return p.ended }
func (p *terminalProcess) attach(cols, rows int) (*TerminalAttachment, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.attachment != nil {
		return nil, workspaceError("", TerminalBusy, errors.New("terminal already attached"))
	}
	if p.ended {
		return nil, workspaceError("", ExecutionFailed, errors.New("shell exited"))
	}
	if p.unsafe {
		return nil, workspaceError("", ExecutionFailed, errors.New("terminal control sequence is incomplete"))
	}
	// Replay at the new geometry, then the daemon signals programs to redraw.
	p.resizeScreen(cols, rows)
	a := &TerminalAttachment{process: p, frames: make(chan []byte, 16), notify: make(chan struct{})}
	if p.seen {
		a.Snapshot = p.snapshot()
	}
	p.attachment = a
	p.answering.Store(false)
	return a, nil
}
func (p *terminalProcess) detach(a *TerminalAttachment) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.attachment == a {
		p.attachment = nil
		p.answering.Store(true)
	}
}
func (p *terminalProcess) input(data []byte) error {
	if len(data) > 16<<10 {
		return errors.New("terminal input frame exceeds its limit")
	}
	select {
	case <-p.done:
		return io.ErrClosedPipe
	default:
	}
	select {
	case p.inputs <- append([]byte(nil), data...):
		return nil
	default:
		return errors.New("terminal input backlog exceeded")
	}
}
func (p *terminalProcess) resize(ctx context.Context, cols, rows int) error {
	p.mu.Lock()
	if p.ended {
		p.mu.Unlock()
		return nil
	}
	p.resizeScreen(cols, rows)
	p.mu.Unlock()
	return p.shell.Resize(ctx, cols, rows)
}

func (p *terminalProcess) resizeScreen(cols, rows int) {
	// The emulator crops bottom rows on shrink. Preserve the prompt and recent
	// output by moving rows above an out-of-view cursor into primary scrollback.
	position := p.screen.CursorPosition()
	if !p.screen.IsAltScreen() && position.Y >= rows {
		shift := position.Y - rows + 1
		width, height := p.screen.Width(), p.screen.Height()
		for y := 0; y < shift; y++ {
			line := make(uv.Line, width)
			for x := 0; x < width; x++ {
				if cell := p.screen.CellAt(x, y); cell != nil {
					line[x] = *cell.Clone()
				}
			}
			p.screen.Scrollback().Push(line)
		}
		for y := 0; y < height-shift; y++ {
			for x := 0; x < width; x++ {
				cell := p.screen.CellAt(x, y+shift)
				if cell != nil {
					cell = cell.Clone()
				}
				p.screen.SetCell(x, y, cell)
			}
		}
	}
	p.screen.Resize(cols, rows)
}
func (p *terminalProcess) stop() error {
	p.once.Do(func() { p.closeErr = p.shell.Close(); <-p.done; p.screen.InputPipe().(io.Closer).Close() })
	return p.closeErr
}
