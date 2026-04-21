// @vitest-environment happy-dom
import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";
import { ref, defineComponent, h } from "vue";
import ButtonRadioGroup from "../../components/shared/ButtonRadioGroup.vue";
import type { ButtonRadioOption } from "../../components/shared/button-radio-group";

// The ButtonRadioGroup is the extracted version of a pattern that used
// to live inline in ExportPdfModal/ExportSettings. Playwright e2e
// selectors (web/e2e/pages/EditorPage.ts) key off `data-page-size`,
// `data-export-style`, and `data-pdf-layout` attributes on the buttons
// — assert the component still emits them. The behavior tests (click
// → emit, v-model, aria-checked) guard the rest of the contract.

describe("ButtonRadioGroup", () => {
  const options: ButtonRadioOption<string>[] = [
    { value: "a", label: "Alpha", dataAttr: { key: "foo", value: "a" } },
    { value: "b", label: "Beta", dataAttr: { key: "foo", value: "b" } },
  ];

  it("renders each option's dataAttr as a data-* attribute", () => {
    const w = mount(ButtonRadioGroup, {
      props: { modelValue: "a", options, ariaLabel: "x" },
    });
    expect(w.find('button[data-foo="a"]').exists()).toBe(true);
    expect(w.find('button[data-foo="b"]').exists()).toBe(true);
  });

  it("marks the selected option aria-checked", () => {
    const w = mount(ButtonRadioGroup, {
      props: { modelValue: "b", options, ariaLabel: "x" },
    });
    expect(w.find('button[data-foo="a"]').attributes("aria-checked")).toBe("false");
    expect(w.find('button[data-foo="b"]').attributes("aria-checked")).toBe("true");
  });

  it("emits update:modelValue on click of a non-selected option", async () => {
    const w = mount(ButtonRadioGroup, {
      props: { modelValue: "a", options, ariaLabel: "x" },
    });
    await w.find('button[data-foo="b"]').trigger("click");
    expect(w.emitted("update:modelValue")).toEqual([["b"]]);
  });

  it("does not emit when clicking the already-selected option", async () => {
    const w = mount(ButtonRadioGroup, {
      props: { modelValue: "a", options, ariaLabel: "x" },
    });
    await w.find('button[data-foo="a"]').trigger("click");
    expect(w.emitted("update:modelValue")).toBeUndefined();
  });

  it("v-models through a parent's ref", async () => {
    const parent = defineComponent({
      setup() {
        const value = ref("a");
        return () =>
          h(ButtonRadioGroup, {
            modelValue: value.value,
            options,
            ariaLabel: "x",
            "onUpdate:modelValue": (v: string) => (value.value = v),
          });
      },
    });
    const w = mount(parent);
    await w.find('button[data-foo="b"]').trigger("click");
    expect(w.find('button[data-foo="b"]').attributes("aria-checked")).toBe("true");
  });

  it("uses flex layout when columns=inline", () => {
    const w = mount(ButtonRadioGroup, {
      props: { modelValue: "a", options, ariaLabel: "x", columns: "inline" as const },
    });
    expect(w.find('[role="radiogroup"]').classes()).toContain("flex");
  });

  it("uses grid layout with the requested column count", () => {
    const w = mount(ButtonRadioGroup, {
      props: { modelValue: "a", options, ariaLabel: "x", columns: 3 },
    });
    const group = w.find('[role="radiogroup"]');
    expect(group.classes()).toContain("grid");
    expect(group.attributes("style")).toContain("repeat(3");
  });
});
