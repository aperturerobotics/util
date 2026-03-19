// MutexLock is a held lock on an AsyncMutex.
export interface MutexLock extends AsyncDisposable {
  // release releases the lock. Idempotent.
  release(): void
  [Symbol.asyncDispose](): Promise<void>
}

// AsyncMutex implements a mutex that accepts an AbortSignal.
// TS translation of the Go csync.Mutex.
export class AsyncMutex {
  private locked = false
  private waiters: Array<() => void> = []

  // lock attempts to hold a lock on the AsyncMutex.
  async lock(signal?: AbortSignal): Promise<MutexLock> {
    signal?.throwIfAborted()

    if (!this.locked) {
      this.locked = true
      return this.newLock()
    }

    return new Promise<MutexLock>((resolve, reject) => {
      const waiter = () => {
        resolve(this.newLock())
      }
      this.waiters.push(waiter)

      if (signal) {
        const onAbort = () => {
          const idx = this.waiters.indexOf(waiter)
          if (idx !== -1) {
            this.waiters.splice(idx, 1)
          }
          reject(signal.reason)
        }
        if (signal.aborted) {
          onAbort()
          return
        }
        signal.addEventListener('abort', onAbort, { once: true })
      }
    })
  }

  // tryLock attempts to hold a lock on the AsyncMutex.
  // Returns a MutexLock or null if the lock could not be grabbed.
  tryLock(): MutexLock | null {
    if (this.locked) {
      return null
    }
    this.locked = true
    return this.newLock()
  }

  private newLock(): MutexLock {
    let released = false
    const release = () => {
      if (released) return
      released = true
      this.locked = false
      const next = this.waiters.shift()
      if (next) {
        this.locked = true
        next()
      }
    }
    return {
      release,
      async [Symbol.asyncDispose]() {
        release()
      },
    }
  }
}
