//go:build unix

package pipesock

import "syscall"

// maxPipePathLen is the longest unix socket path this platform binds.
// sun_path is a fixed-size field and the kernel requires a terminating zero
// byte inside it, so one byte fewer than the field is usable.
var maxPipePathLen = len(syscall.RawSockaddrUnix{}.Path) - 1
