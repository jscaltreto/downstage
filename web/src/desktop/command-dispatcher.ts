// The frontend command dispatcher. One source of truth for:
//
// - Executing commands by ID (menu event → dispatcher → handler).
// - Deciding which commands are currently enabled.
// - Telling Go which IDs to grey out on the native menu.
//
// Commands are registered with an optional `isEnabled` predicate. A
// disabled dispatch is a silent no-op — the surface layer (menu item,
// palette row) is responsible for showing the user that the command
// isn't available. Handlers themselves never police preconditions; if
// they're invoked, they run.
//
// Menu reflection: the dispatcher keeps a `lastDisabledIds` snapshot.
// When the caller asks it to refresh (via `refreshDisabledSet`), it
// recomputes, diffs, and only pushes to Go on a real change. A
// single-flight microtask guard batches bursts of reactive updates so
// a multi-field state mutation produces at most one Go round-trip.

export type CommandHandler = () => void | Promise<void>;

export interface HandlerEntry {
  handler: CommandHandler;
  isEnabled?: () => boolean;
}

export interface DispatcherBackend {
  // Push the latest disabled-id set to the host menu. The dispatcher
  // will not call this unless the set actually changed vs. the last
  // call — backends can treat every invocation as a real change.
  setDisabledCommands(ids: string[]): Promise<void>;
}

export class CommandDispatcher {
  private handlers = new Map<string, HandlerEntry>();
  private lastDisabledIds: Set<string> = new Set();
  private pendingFlush = false;

  constructor(private backend: DispatcherBackend) {}

  register(id: string, entry: HandlerEntry): void {
    this.handlers.set(id, entry);
  }

  async dispatch(id: string): Promise<void> {
    const entry = this.handlers.get(id);
    if (!entry) {
      // Unknown ID — silent no-op. Menu items are declared in Go; a TS
      // handler gap would be a bug, but it shouldn't crash the app.
      console.warn(`command dispatcher: no handler registered for "${id}"`);
      return;
    }
    if (entry.isEnabled && !entry.isEnabled()) {
      // Disabled — silent drop. Surfaces above have already signalled
      // unavailability.
      return;
    }
    await entry.handler();
  }

  disabledIds(): string[] {
    return Array.from(this.lastDisabledIds);
  }

  // scheduleRefresh batches reactive invalidations from Vue watchers into
  // a single microtask flush. Callers don't need to debounce themselves.
  scheduleRefresh(): void {
    if (this.pendingFlush) return;
    this.pendingFlush = true;
    queueMicrotask(() => {
      this.pendingFlush = false;
      void this.refresh();
    });
  }

  // refresh recomputes the disabled set, diffs against the last pushed
  // snapshot, and forwards to the backend only on change. Exposed so
  // tests can drive it deterministically without waiting on microtasks.
  async refresh(): Promise<void> {
    const next = new Set<string>();
    for (const [id, entry] of this.handlers) {
      if (entry.isEnabled && !entry.isEnabled()) {
        next.add(id);
      }
    }
    if (setsEqual(next, this.lastDisabledIds)) return;
    this.lastDisabledIds = next;
    await this.backend.setDisabledCommands(Array.from(next));
  }
}

function setsEqual(a: Set<string>, b: Set<string>): boolean {
  if (a.size !== b.size) return false;
  for (const v of a) if (!b.has(v)) return false;
  return true;
}
