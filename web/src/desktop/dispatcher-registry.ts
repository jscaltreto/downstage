// Tiny registry allowing `desktop-app.ts` (which owns the Wails event
// subscription) to reach the live CommandDispatcher instance that
// `AppDesktop.vue` constructs in `onMounted`. Mirrors the flush-save
// registry pattern — the Vue component side owns lifecycle; the
// module-scope side owns the subscription.
//
// AppDesktop's onMounted does `await store.init(); await
// workspace.init();` before constructing + registering the dispatcher
// — tens to hundreds of ms during which the native menu is already
// interactive. Without the buffer below, a menu click during that
// window is silently dropped. The buffer collects ids until the
// dispatcher arrives, then drains in order.

import type { CommandDispatcher } from "./command-dispatcher";

let current: CommandDispatcher | null = null;
let pending: string[] = [];

export function registerDispatcher(dispatcher: CommandDispatcher | null) {
  current = dispatcher;
  if (!dispatcher) {
    // Unregister (component unmount). Don't clear the queue — the
    // next mount will drain whatever accumulated.
    return;
  }
  // Drain in arrival order. Snapshot first so a handler that
  // dispatches synchronously (unlikely but cheap to guard) can't
  // grow the queue mid-drain.
  const toDispatch = pending;
  pending = [];
  for (const id of toDispatch) {
    void dispatcher.dispatch(id);
  }
}

export async function dispatchCommand(id: string): Promise<void> {
  if (!current) {
    pending.push(id);
    return;
  }
  await current.dispatch(id);
}

// Test-only: reset module state between test cases. The registry's
// module-scope `current` and `pending` would otherwise persist across
// tests in the same suite.
export function _resetDispatcherRegistryForTests() {
  current = null;
  pending = [];
}
