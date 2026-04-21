// Shape for ButtonRadioGroup options. Lives in a sibling .ts file so
// that consumers can `import type { ButtonRadioOption } from …` without
// fighting Vue's <script setup> type-export rules (TS-as-host can't
// re-export named types out of a generic SFC).

export interface ButtonRadioOption<T> {
  value: T;
  label: string;
  // Optional { key, value } pair rendered as a `data-<key>="<value>"`
  // attribute on the button. Lets callers keep Playwright locators like
  // `button[data-pdf-layout="single"]` working after extraction.
  dataAttr?: { key: string; value: string };
}
