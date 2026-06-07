// @vitest-environment happy-dom
import { describe, expect, it, vi } from "vitest";
import { mount, flushPromises } from "@vue/test-utils";
import ReviewChangesModal from "../../desktop/ReviewChangesModal.vue";
import type { LibraryDirty } from "../../desktop/types";

// BaseModal renders inside a teleport, which @vue/test-utils handles
// poorly out of the box. Stub it to a transparent wrapper so we can
// query the rendered children directly.
const baseModalStub = {
  name: "BaseModal",
  props: ["open", "title"],
  emits: ["close"],
  template: '<div v-if="open" data-testid="modal-root"><slot /></div>',
};

const promptModalStub = {
  name: "PromptModal",
  props: ["open", "title", "label", "initialValue", "submitLabel"],
  emits: ["close", "submit"],
  template: '<div v-if="open" data-testid="prompt-stub"></div>',
};

const confirmModalStub = {
  name: "ConfirmModal",
  props: ["open", "title", "message", "confirmLabel", "destructive"],
  emits: ["close", "confirm"],
  template: '<div v-if="open" data-testid="confirm-stub"></div>',
};

function makeDirty(): LibraryDirty {
  return {
    plays: [
      { path: "act-one.ds", kind: "modified" },
      { path: "act-two.ds", kind: "untracked" },
    ],
    sidecars: [
      { path: ".downstage/dictionary.txt", kind: "modified" },
    ],
    other: [],
    count: 3,
  };
}

function mountModal(props: Partial<{ open: boolean; dirty: LibraryDirty | null; busy: boolean }> = {}) {
  return mount(ReviewChangesModal, {
    props: {
      open: true,
      dirty: makeDirty(),
      busy: false,
      ...props,
    },
    global: {
      stubs: {
        BaseModal: baseModalStub,
        PromptModal: promptModalStub,
        ConfirmModal: confirmModalStub,
      },
    },
  });
}

describe("ReviewChangesModal", () => {
  it("renders one section per non-empty category", async () => {
    const wrapper = mountModal();
    await flushPromises();
    const headers = wrapper.findAll("h4").map((h) => h.text());
    // Plays (2) + Library settings (1). Other was empty, so no section.
    expect(headers.length).toBe(2);
    expect(headers[0]).toContain("Plays");
    expect(headers[1]).toContain("Library settings");
  });

  it("does not render empty categories", async () => {
    const wrapper = mountModal({
      dirty: { plays: [{ path: "a.ds", kind: "modified" }], sidecars: [], other: [], count: 1 },
    });
    await flushPromises();
    const headers = wrapper.findAll("h4").map((h) => h.text());
    expect(headers.length).toBe(1);
    expect(headers[0]).toContain("Plays");
  });

  it("commit-all opens the prompt with all paths queued, then emits on submit", async () => {
    const wrapper = mountModal();
    await flushPromises();
    const commitAllBtn = wrapper.findAll("button").find((b) => b.text().includes("Commit all changes"));
    expect(commitAllBtn).toBeTruthy();
    await commitAllBtn!.trigger("click");
    await flushPromises();

    // Prompt stub should now be open.
    const prompt = wrapper.findComponent(promptModalStub as any);
    expect(prompt.props("open")).toBe(true);

    prompt.vm.$emit("submit", "Library cleanup commit");
    await flushPromises();

    const emitted = wrapper.emitted("commit");
    expect(emitted, "commit event should fire").toBeTruthy();
    expect(emitted![0]).toEqual([
      ["act-one.ds", "act-two.ds", ".downstage/dictionary.txt"],
      "Library cleanup commit",
    ]);
  });

  it("per-section commit only includes that section's paths", async () => {
    const wrapper = mountModal();
    await flushPromises();
    // First "Commit all" button is the Plays section header button.
    const sectionCommit = wrapper.findAll("button").find((b) => b.text().trim().startsWith("Commit all"));
    expect(sectionCommit).toBeTruthy();
    await sectionCommit!.trigger("click");
    await flushPromises();

    const prompt = wrapper.findComponent(promptModalStub as any);
    expect(prompt.props("open")).toBe(true);
    prompt.vm.$emit("submit", "Update plays");
    await flushPromises();

    expect(wrapper.emitted("commit")![0]).toEqual([
      ["act-one.ds", "act-two.ds"],
      "Update plays",
    ]);
  });

  it("discard buttons open a styled confirm modal and emit only on confirm", async () => {
    const wrapper = mountModal();
    await flushPromises();

    // Section "Discard all" should open the confirm modal — but not emit yet.
    const sectionDiscard = wrapper.findAll("button").find((b) => b.text().trim().startsWith("Discard all"));
    await sectionDiscard!.trigger("click");
    await flushPromises();
    expect(wrapper.emitted("discard")).toBeFalsy();

    const confirmStub = wrapper.findComponent(confirmModalStub as any);
    expect(confirmStub.props("open")).toBe(true);
    // `destructive` is bound as a bare attribute, which the stub receives
    // as "". Just confirm the prop was passed at all.
    expect(confirmStub.props()).toHaveProperty("destructive");

    // Closing the modal cancels — no emit.
    confirmStub.vm.$emit("close");
    await flushPromises();
    expect(wrapper.emitted("discard")).toBeFalsy();

    // Re-open and confirm — the paths flow through.
    await sectionDiscard!.trigger("click");
    await flushPromises();
    confirmStub.vm.$emit("confirm");
    await flushPromises();
    expect(wrapper.emitted("discard")![0]).toEqual([["act-one.ds", "act-two.ds"]]);
  });

  it("commit-all is disabled when the library is clean", async () => {
    const wrapper = mountModal({
      dirty: { plays: [], sidecars: [], other: [], count: 0 },
    });
    await flushPromises();
    const commitAllBtn = wrapper.findAll("button").find((b) => b.text().includes("Commit all changes"));
    expect(commitAllBtn!.attributes("disabled")).toBeDefined();
  });

  it("emits close on Close button", async () => {
    const wrapper = mountModal();
    await flushPromises();
    const closeBtn = wrapper.findAll("button").find((b) => b.text().trim() === "Close");
    await closeBtn!.trigger("click");
    expect(wrapper.emitted("close")).toBeTruthy();
  });

  it("emits refresh on Refresh button", async () => {
    const wrapper = mountModal();
    await flushPromises();
    const refresh = wrapper.findAll("button").find((b) => b.text().includes("Refresh"));
    await refresh!.trigger("click");
    expect(wrapper.emitted("refresh")).toBeTruthy();
  });
});
