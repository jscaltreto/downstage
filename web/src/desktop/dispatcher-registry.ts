// Tiny registry allowing `desktop-app.ts` (which owns the Wails event
// subscription) to reach the live CommandDispatcher instance that
// `AppDesktop.vue` constructs in `onMounted`. Mirrors the flush-save
// registry pattern — the Vue component side owns lifecycle; the
// module-scope side owns the subscription.

import type { CommandDispatcher } from "./command-dispatcher";

let current: CommandDispatcher | null = null;

export function registerDispatcher(dispatcher: CommandDispatcher | null) {
  current = dispatcher;
}

export async function dispatchCommand(id: string): Promise<void> {
  if (!current) {
    // Command event fired before the Vue side finished mounting. Rare
    // but not fatal — log and drop.
    console.warn(`command dispatcher not yet registered; dropping "${id}"`);
    return;
  }
  await current.dispatch(id);
}
