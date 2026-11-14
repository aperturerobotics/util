//go:build windows

package pipesock

import (
	"context"
	"errors"
	"net"
	"os"
	"sync"

	"github.com/Microsoft/go-winio"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
)

// pipeBufferSize is the in and out buffer size of each pipe instance.
const pipeBufferSize = 64 * 1024

// BuildPipeListener builds the pipe listener in the directory.
func BuildPipeListener(le *logrus.Entry, rootDir, pipeUuid string) (net.Listener, error) {
	pipeName := BuildPipeName(rootDir, pipeUuid)
	le.Debugf("listening on named pipe: %s", pipeName)
	return listenPipe(pipeName)
}

// DialPipeListener connects to the pipe listener in the directory.
func DialPipeListener(ctx context.Context, le *logrus.Entry, rootDir, pipeUuid string) (net.Conn, error) {
	pipeName := BuildPipeName(rootDir, pipeUuid)
	le.Debugf("connecting to named pipe: %s", pipeName)
	return winio.DialPipeContext(ctx, pipeName)
}

// BuildPipeName builds a unique pipe name from a path and uuid.
// uuid must be unique for rootDir
func BuildPipeName(rootDir, pipeUuid string) string {
	return `\\.\pipe\aptre\` + pipeUuid
}

// pipeListener accepts connections on a named pipe.
//
// It keeps one instance listening at all times, created with read and write
// access, so a client that opens the name connects to an instance Accept
// will hand out, even before Accept is called. Once a client connects, the
// next instance is created before the connection is returned.
//
// It does not reserve the name with an instance that has neither read nor
// write access, as winio.ListenPipe does. Windows keeps clients off such an
// instance, but Wine connects them to it, where nobody ever reads.
type pipeListener struct {
	name   string
	name16 *uint16
	// closeEv is set by Close to wake a waiting Accept.
	closeEv windows.Handle

	closeOnce sync.Once

	// mtx serializes Accept and Close and guards the fields below.
	mtx sync.Mutex
	// closed is set once Close starts.
	closed bool
	// pending is the listening instance, or 0 after a failed create.
	pending windows.Handle
}

// listenPipe creates the first instance of the named pipe. On Windows it
// fails if another listener already owns the name; Wine does not check.
func listenPipe(name string) (*pipeListener, error) {
	name16, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, &os.PathError{Op: "listen", Path: name, Err: err}
	}
	closeEv, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return nil, err
	}
	l := &pipeListener{name: name, name16: name16, closeEv: closeEv}
	l.pending, err = l.createInstance(true)
	if err != nil {
		_ = windows.CloseHandle(closeEv)
		return nil, err
	}
	return l, nil
}

// createInstance creates a listening instance of the pipe. The instance
// accepts a client as soon as it exists.
func (l *pipeListener) createInstance(first bool) (windows.Handle, error) {
	flags := uint32(windows.PIPE_ACCESS_DUPLEX | windows.FILE_FLAG_OVERLAPPED)
	if first {
		flags |= windows.FILE_FLAG_FIRST_PIPE_INSTANCE
	}
	mode := uint32(windows.PIPE_TYPE_BYTE | windows.PIPE_READMODE_BYTE | windows.PIPE_WAIT | windows.PIPE_REJECT_REMOTE_CLIENTS)
	h, err := windows.CreateNamedPipe(l.name16, flags, mode, windows.PIPE_UNLIMITED_INSTANCES, pipeBufferSize, pipeBufferSize, 0, nil)
	if err != nil {
		return 0, &os.PathError{Op: "listen", Path: l.name, Err: err}
	}
	return h, nil
}

// Accept waits for a client to connect and returns the connection.
// A client that connects and leaves before Accept sees it is skipped.
func (l *pipeListener) Accept() (net.Conn, error) {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	for {
		if l.closed {
			return nil, net.ErrClosed
		}
		if l.pending == 0 {
			h, err := l.createInstance(false)
			if err != nil {
				return nil, err
			}
			l.pending = h
		}

		h := l.pending
		err := l.connect(h)
		if errors.Is(err, net.ErrClosed) {
			return nil, err
		}
		if err != nil {
			// The client left before the connection completed, or the
			// instance broke: discard it and listen on a fresh one.
			_ = windows.CloseHandle(h)
			l.pending = 0
			continue
		}

		// Listen on the next instance before handing this one out, so the
		// name stays open to the next client. On failure the next Accept
		// retries the create.
		l.pending, _ = l.createInstance(false)

		// NewFile detects the overlapped handle and adds it to the runtime
		// poller, which gives the connection deadlines and a Close that
		// unblocks pending reads and writes.
		return &pipeConn{File: os.NewFile(uintptr(h), l.name), addr: pipeAddr(l.name)}, nil
	}
}

// connect waits for a client on the instance. It returns net.ErrClosed when
// Close interrupts the wait.
func (l *pipeListener) connect(h windows.Handle) error {
	ev, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(ev) //nolint:errcheck

	ov := &windows.Overlapped{HEvent: ev}
	switch err := windows.ConnectNamedPipe(h, ov); {
	case err == nil, errors.Is(err, windows.ERROR_PIPE_CONNECTED):
		return nil
	case !errors.Is(err, windows.ERROR_IO_PENDING):
		return err
	}

	var n uint32
	which, err := windows.WaitForMultipleObjects([]windows.Handle{ev, l.closeEv}, false, windows.INFINITE)
	if err == nil && which == windows.WAIT_OBJECT_0 {
		return windows.GetOverlappedResult(h, ov, &n, false)
	}

	// Close was called or the wait failed: cancel the connect and wait for
	// it to finish, since the kernel writes ov until then.
	_ = windows.CancelIoEx(h, ov)
	_ = windows.GetOverlappedResult(h, ov, &n, true)
	if err != nil {
		return err
	}
	return net.ErrClosed
}

// Close stops listening. A waiting Accept returns net.ErrClosed; connections
// already accepted stay open.
func (l *pipeListener) Close() error {
	l.closeOnce.Do(func() {
		_ = windows.SetEvent(l.closeEv)
		l.mtx.Lock()
		defer l.mtx.Unlock()
		l.closed = true
		if l.pending != 0 {
			_ = windows.CloseHandle(l.pending)
			l.pending = 0
		}
		_ = windows.CloseHandle(l.closeEv)
	})
	return nil
}

// Addr returns the pipe name.
func (l *pipeListener) Addr() net.Addr {
	return pipeAddr(l.name)
}

// pipeAddr is the name of a named pipe.
type pipeAddr string

// Network returns "pipe".
func (pipeAddr) Network() string { return "pipe" }

// String returns the pipe name.
func (a pipeAddr) String() string { return string(a) }

// pipeConn is the server end of a connected pipe instance.
type pipeConn struct {
	*os.File
	addr pipeAddr
}

// LocalAddr returns the pipe name.
func (c *pipeConn) LocalAddr() net.Addr { return c.addr }

// RemoteAddr returns the pipe name.
func (c *pipeConn) RemoteAddr() net.Addr { return c.addr }

// _ is a type assertion
var (
	_ net.Listener = (*pipeListener)(nil)
	_ net.Conn     = (*pipeConn)(nil)
)
