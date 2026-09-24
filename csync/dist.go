//go:build !js && !tinygo

package csync

import "embed"

// DistSources contains the TypeScript RWMutex for the web.
//
//go:embed rwmutex.ts
var DistSources embed.FS
