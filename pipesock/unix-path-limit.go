//go:build !windows && !js && !wasip1 && !plan9

package pipesock

import (
	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

const maxUnixSocketPathLength = len(unix.RawSockaddrUnix{}.Path)

func validateUnixSocketPath(path string) error {
	if len(path) > maxUnixSocketPathLength {
		return errors.Errorf(
			"unix socket path exceeds %d-byte limit: path is %d bytes: %s",
			maxUnixSocketPathLength,
			len(path),
			path,
		)
	}
	return nil
}
