//go:build !js && !tinygo

package backoff

import "embed"

// DistSources contains the TypeScript backoff messages for the web.
//
//go:embed backoff.pb.ts
var DistSources embed.FS
