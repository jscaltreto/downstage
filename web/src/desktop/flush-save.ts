// Module-scoped registry for the active flushSave function. The Wails
// `downstage:before-close` listener lives in `desktop-app.ts` (which is
// desktop-build-only because it imports generated Wails bindings); this
// shim lets `AppDesktop.vue` register/unregister its flush function
// without reaching into desktop-app.ts directly, keeping the component
// importable from tests that don't have the Wails runtime available.

let flushSaveRef: (() => Promise<void>) | null = null;

export function registerFlushSave(fn: (() => Promise<void>) | null): void {
  flushSaveRef = fn;
}

export async function invokeRegisteredFlushSave(): Promise<void> {
  if (flushSaveRef) await flushSaveRef();
}
