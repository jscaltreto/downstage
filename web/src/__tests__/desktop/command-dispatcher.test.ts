import { describe, expect, it, vi } from "vitest";
import {
  CommandDispatcher,
  type DispatcherBackend,
} from "../../desktop/command-dispatcher";

function makeBackend() {
  const sets: string[][] = [];
  const backend: DispatcherBackend = {
    setDisabledCommands: vi.fn(async (ids: string[]) => {
      sets.push([...ids].sort());
    }),
  };
  return { backend, sets };
}

describe("CommandDispatcher", () => {
  it("dispatch runs the registered handler", async () => {
    const { backend } = makeBackend();
    const d = new CommandDispatcher(backend);
    const handler = vi.fn();
    d.register("x", { handler });

    await d.dispatch("x");
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("dispatch on an unknown id is a silent no-op (warns, does not throw)", async () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const { backend } = makeBackend();
    const d = new CommandDispatcher(backend);
    await expect(d.dispatch("missing")).resolves.toBeUndefined();
    expect(warn).toHaveBeenCalled();
    warn.mockRestore();
  });

  it("dispatch on a disabled command is a silent no-op", async () => {
    const { backend } = makeBackend();
    const d = new CommandDispatcher(backend);
    const handler = vi.fn();
    d.register("x", { handler, isEnabled: () => false });

    await d.dispatch("x");
    expect(handler).not.toHaveBeenCalled();
  });

  it("refresh pushes the disabled set on real change", async () => {
    const { backend } = makeBackend();
    const d = new CommandDispatcher(backend);
    let enabled = true;
    d.register("x", { handler: () => {}, isEnabled: () => enabled });

    await d.refresh();
    // Initially enabled — disabled set is empty. First push is []
    // because this IS a change vs. the "never pushed" baseline (empty
    // set), but setsEqual short-circuits equal sets. Verify via
    // `sets.length` (push count) that nothing went over the wire.
    expect(backend.setDisabledCommands).not.toHaveBeenCalled();

    enabled = false;
    await d.refresh();
    expect(backend.setDisabledCommands).toHaveBeenCalledTimes(1);
    expect(backend.setDisabledCommands).toHaveBeenLastCalledWith(["x"]);
  });

  it("diff-and-skip: two unrelated refreshes that keep the set stable do NOT push", async () => {
    const { backend } = makeBackend();
    const d = new CommandDispatcher(backend);
    let theme = "light";
    // isEnabled depends on `theme` but the result is always true. A
    // state mutation shouldn't force a wire push if the disabled set
    // didn't actually change.
    d.register("x", { handler: () => {}, isEnabled: () => theme.length > 0 });

    await d.refresh(); // baseline
    theme = "dark";
    await d.refresh();
    theme = "system";
    await d.refresh();

    expect(backend.setDisabledCommands).not.toHaveBeenCalled();
  });

  it("single-flight microtask flush collapses a burst into one refresh", async () => {
    const { backend } = makeBackend();
    const d = new CommandDispatcher(backend);
    let enabled = true;
    d.register("x", { handler: () => {}, isEnabled: () => enabled });

    // Simulate a burst of reactive updates in the same tick. Multiple
    // scheduleRefresh calls should collapse into one microtask.
    enabled = false;
    d.scheduleRefresh();
    d.scheduleRefresh();
    d.scheduleRefresh();

    // Yield the microtask queue.
    await Promise.resolve();
    await Promise.resolve();

    expect(backend.setDisabledCommands).toHaveBeenCalledTimes(1);
    expect(backend.setDisabledCommands).toHaveBeenLastCalledWith(["x"]);
  });
});
