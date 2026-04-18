// PrefsCache owns the authoritative in-memory copy of the desktop
// Preferences struct and serializes writes back to the backend. It exists
// so multiple independent writers (Store for editor prefs, Workspace for
// sidebar) can't interleave read-modify-write cycles against the backend
// and clobber each other's fields.
//
// Semantics:
// - Reads return the cached snapshot. The first read lazy-loads from the
//   backend; concurrent first-reads share the same promise.
// - Writes mutate the snapshot synchronously (within the same task) and
//   enqueue a backend write at the tail of a promise chain. Every queued
//   write captures the snapshot-at-enqueue-time, so distinct writes
//   produce distinct on-disk writes even when they arrive back-to-back.
// - `flush()` resolves once the tail of the write chain has completed.
//   Callers should await it on app shutdown to avoid dropping in-flight
//   writes.
// - A failed write does not poison the chain — subsequent writes still
//   run. The failure is surfaced through the returned promise of that
//   specific `update` call so the caller can log it.

export interface PrefsBackend<T extends object> {
  load(): Promise<T>;
  save(prefs: T): Promise<void>;
}

export interface PrefsCache<T extends object> {
  get(): Promise<T>;
  update(patch: Partial<T>): Promise<void>;
  flush(): Promise<void>;
}

export function createPrefsCache<T extends object>(
  backend: PrefsBackend<T>,
): PrefsCache<T> {
  let snapshot: T | null = null;
  let loaded: Promise<void> | null = null;
  // `chain` is the serialization primitive. Every update appends to its
  // tail; `flush` awaits the tail. `.catch(() => {})` keeps a failure from
  // making the chain permanently rejected — individual writes still
  // surface their own errors via the promise they return.
  let chain: Promise<void> = Promise.resolve();

  async function ensureLoaded(): Promise<void> {
    if (!loaded) {
      loaded = (async () => {
        snapshot = await backend.load();
      })();
    }
    return loaded;
  }

  return {
    async get(): Promise<T> {
      await ensureLoaded();
      // Return a shallow copy so callers can't mutate the authoritative
      // snapshot from outside.
      return { ...(snapshot as T) };
    },

    update(patch: Partial<T>): Promise<void> {
      // Enqueue synchronously so `flush()` sees this write even if the
      // first load is still in flight. The actual mutation and save
      // happen inside the chain so they remain strictly serialized.
      const next = chain.catch(() => {}).then(async () => {
        await ensureLoaded();
        Object.assign(snapshot as T, patch);
        const toWrite: T = { ...(snapshot as T) };
        await backend.save(toWrite);
      });
      chain = next;
      return next;
    },

    async flush(): Promise<void> {
      // Swallow errors here — flush is a "wait for quiescence" primitive,
      // not an error sink. Individual update() callers have already seen
      // their own errors.
      await chain.catch(() => {});
    },
  };
}
