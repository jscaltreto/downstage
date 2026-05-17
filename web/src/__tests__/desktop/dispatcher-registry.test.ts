import { describe, it, expect, beforeEach, vi } from "vitest";

import {
  dispatchCommand,
  registerDispatcher,
  _resetDispatcherRegistryForTests,
} from "../../desktop/dispatcher-registry";
import type { CommandDispatcher } from "../../desktop/command-dispatcher";

// M8: AppDesktop.vue's onMounted runs `await store.init(); await
// workspace.init();` BEFORE constructing + registering the
// CommandDispatcher. That's tens to hundreds of ms of env reads during
// which the native menu is interactive but the dispatcher is null.
// Without a queue, every menu click in that window is silently dropped.

function makeFakeDispatcher() {
  const calls: string[] = [];
  const dispatcher: CommandDispatcher = {
    dispatch: vi.fn(async (id: string) => {
      calls.push(id);
    }),
  } as unknown as CommandDispatcher;
  return { dispatcher, calls };
}

describe("dispatcher-registry", () => {
  beforeEach(() => {
    _resetDispatcherRegistryForTests();
  });

  it("dispatchCommand before register buffers; register drains in order", async () => {
    // Three commands fired during the startup window.
    void dispatchCommand("file.newPlay");
    void dispatchCommand("view.commandPalette");
    void dispatchCommand("file.settings");

    const { dispatcher, calls } = makeFakeDispatcher();
    registerDispatcher(dispatcher);

    // The register call drains synchronously (fire-and-forget); the
    // dispatch promises resolve on the next microtask.
    await Promise.resolve();
    await Promise.resolve();

    expect(calls).toEqual([
      "file.newPlay",
      "view.commandPalette",
      "file.settings",
    ]);
  });

  it("dispatchCommand after register goes through immediately", async () => {
    const { dispatcher, calls } = makeFakeDispatcher();
    registerDispatcher(dispatcher);

    await dispatchCommand("file.saveVersion");

    expect(calls).toEqual(["file.saveVersion"]);
  });

  it("dispatchCommand with no dispatcher and never-registered drops nothing on the floor on re-register", async () => {
    // Drop a command, unregister (e.g. unmount during dev HMR), drop
    // another, then re-register. Both should drain.
    void dispatchCommand("file.newPlay");
    registerDispatcher(null);
    void dispatchCommand("view.commandPalette");

    const { dispatcher, calls } = makeFakeDispatcher();
    registerDispatcher(dispatcher);

    await Promise.resolve();
    await Promise.resolve();

    expect(calls).toContain("file.newPlay");
    expect(calls).toContain("view.commandPalette");
  });

  it("registerDispatcher(null) does not throw and preserves the pending queue", () => {
    void dispatchCommand("file.newPlay");
    expect(() => registerDispatcher(null)).not.toThrow();
  });
});
