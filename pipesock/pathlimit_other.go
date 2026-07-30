//go:build !unix && !windows

package pipesock

// maxPipePathLen is the shortest sun_path limit among the unix platforms,
// used where the platform does not expose sun_path at all.
var maxPipePathLen = 103
