import { describe, it, expect } from 'vitest'
import path from 'path'
import { buildPipeName } from './pipesock.js'

const itUnix = it.skipIf(process.platform === 'win32')

describe('buildPipeName', () => {
  itUnix('builds a socket path under the platform limit', () => {
    const pipePath = buildPipeName('/tmp', 'startup')
    expect(path.basename(pipePath)).toBe('.pipe-startup')
  })

  itUnix('rejects an over-long unix socket path by name', () => {
    // Exceeds every sun_path limit both as an absolute path and as a path
    // relative to the working directory, mirroring pathlimit_test.go.
    let root = '/tmp'
    for (let i = 0; i < 8; i++) {
      root = path.join(root, 'd'.repeat(24))
    }
    expect(() => buildPipeName(root, 'startup')).toThrowError(
      /unix socket path is \d+ bytes and this platform binds at most \d+: .*\.pipe-startup/,
    )
  })
})
