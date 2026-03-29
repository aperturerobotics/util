package pipesock

import (
	"embed"
)

// DistSources contains the sources for the web.
//
//go:embed pipesock.ts
var DistSources embed.FS
