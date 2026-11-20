//go:build !windows

// Package pipesock opens a local socket between processes on one device.
//
// On Unix, BuildPipeListener listens on a socket file named by the pipe ID
// under a root directory, and DialPipeListener connects to it. The path uses
// the shorter of the absolute and the working-directory-relative form, and a
// path longer than the kernel's sun_path limit is rejected by name. On
// Windows the pipe ID names a named pipe instead.
package pipesock

import (
	"context"
	"net"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// resolvePipePath builds the unix socket path for a pipe.
// The kernel bounds sun_path, so an over-long path is rejected here by name
// rather than at bind time as an opaque invalid argument.
func resolvePipePath(le *logrus.Entry, rootDir, pipeUuid string) (string, error) {
	// Create absolute path for the socket
	absolutePipePath := filepath.Join(rootDir, ".pipe-"+pipeUuid)

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		// If we can't get CWD (e.g., it was deleted), use absolute path
		le.WithError(err).Debug("could not get cwd, using absolute pipe path")
		cwd = ""
	}

	// Get relative path from current working directory if possible
	// Use whichever is shorter, since sun_path is small.
	pipePath := absolutePipePath
	if cwd != "" {
		relPath, err := filepath.Rel(cwd, absolutePipePath)
		if err == nil && len(relPath) < len(pipePath) {
			pipePath = relPath
		}
	}

	if len(pipePath) > maxPipePathLen {
		return "", errors.Errorf(
			"unix socket path is %d bytes and this platform binds at most %d: %s",
			len(pipePath), maxPipePathLen, pipePath,
		)
	}
	return pipePath, nil
}

// BuildPipeListener builds the pipe listener.
// The rootDir is used for unix sockets if this is a linux system.
// The pipeUuid is used for the socket path OR the Windows Pipe Name.
// The pipeUuid should be unique to the local device and pipe.
func BuildPipeListener(le *logrus.Entry, rootDir, pipeUuid string) (net.Listener, error) {
	// Ensure the parent directory exists
	pipeDir := filepath.Dir(filepath.Join(rootDir, ".pipe-"+pipeUuid))
	if err := os.MkdirAll(pipeDir, 0o755); err != nil {
		return nil, errors.Wrap(err, "create pipe directory")
	}

	pipePath, err := resolvePipePath(le, rootDir, pipeUuid)
	if err != nil {
		return nil, err
	}

	// remove old pipe file, if exists
	if _, err := os.Stat(pipePath); !os.IsNotExist(err) {
		if err := os.Remove(pipePath); err != nil {
			return nil, errors.Wrap(err, "remove old pipe file")
		}
	}

	addr := &net.UnixAddr{
		Net:  "unix",
		Name: pipePath,
	}
	le.Debugf("listening on unix socket: %s", addr.String())
	return net.ListenUnix("unix", addr)
}

// DialPipeListener connects to the pipe listener in the directory.
func DialPipeListener(ctx context.Context, le *logrus.Entry, rootDir, pipeUuid string) (net.Conn, error) {
	pipePath, err := resolvePipePath(le, rootDir, pipeUuid)
	if err != nil {
		return nil, err
	}

	addr := &net.UnixAddr{
		Net:  "unix",
		Name: pipePath,
	}
	le.Debugf("connecting to unix socket: %s", addr.String())
	dialer := net.Dialer{}
	return dialer.DialContext(ctx, "unix", addr.String())
}
