// RWMutexLock is a held lock on an AsyncRWMutex.
export interface RWMutexLock extends AsyncDisposable {
  // release releases the lock. Idempotent.
  release(): void
  [Symbol.asyncDispose](): Promise<void>
}

// AsyncRWMutex implements an RWMutex that accepts an AbortSignal.
// A single writer OR many readers can hold Lock at a time.
// If a writer is waiting to lock, readers will wait for it (write-preferring).
// TS translation of the Go csync.RWMutex.
export class AsyncRWMutex {
  private nreaders = 0
  private writing = false
  private writeWaiting = 0
  private waiters: Array<{ write: boolean; resolve: () => void }> = []

  // lock attempts to hold a lock on the AsyncRWMutex.
  async lock(write: boolean, signal?: AbortSignal): Promise<RWMutexLock> {
    signal?.throwIfAborted()

    if (this.canAcquire(write)) {
      this.acquire(write)
      return this.newLock(write)
    }

    if (write) {
      this.writeWaiting++
    }

    return new Promise<RWMutexLock>((resolve, reject) => {
      const entry = {
        write,
        resolve: () => {
          resolve(this.newLock(write))
        },
      }
      this.waiters.push(entry)

      if (signal) {
        const onAbort = () => {
          const idx = this.waiters.indexOf(entry)
          if (idx !== -1) {
            this.waiters.splice(idx, 1)
            if (write) {
              this.writeWaiting--
              this.wake()
            }
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

  // tryLock attempts to hold a lock on the AsyncRWMutex.
  // Returns a RWMutexLock or null if the lock could not be grabbed.
  tryLock(write: boolean): RWMutexLock | null {
    if (!this.canAcquire(write)) {
      return null
    }
    this.acquire(write)
    return this.newLock(write)
  }

  private canAcquire(write: boolean): boolean {
    if (write) {
      return this.nreaders === 0 && !this.writing
    }
    return !this.writing && this.writeWaiting === 0
  }

  private acquire(write: boolean): void {
    if (write) {
      this.writing = true
    } else {
      this.nreaders++
    }
  }

  private newLock(write: boolean): RWMutexLock {
    let released = false
    const release = () => {
      if (released) return
      released = true
      if (write) {
        this.writing = false
      } else {
        this.nreaders--
      }
      this.wake()
    }
    return {
      release,
      async [Symbol.asyncDispose]() {
        release()
      },
    }
  }

  private wake(): void {
    // Wake eligible waiters. Writers get priority.
    const pending = [...this.waiters]
    for (const entry of pending) {
      if (this.canAcquire(entry.write)) {
        const idx = this.waiters.indexOf(entry)
        if (idx !== -1) {
          this.waiters.splice(idx, 1)
          if (entry.write) {
            this.writeWaiting--
          }
          this.acquire(entry.write)
          entry.resolve()
          // If we just acquired a write lock, stop waking.
          if (entry.write) return
        }
      }
    }
  }
}
