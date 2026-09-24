//go:build !js && !tinygo

package filter

import "embed"

// DistSources contains the TypeScript filter messages for the web.
//
//go:embed filter.pb.ts
var DistSources embed.FS
