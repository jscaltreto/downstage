import type { EditorView } from "@codemirror/view";
import { renderPDF } from "./wasm";

function extractDraftTitle(content: string): string {
  const match = content.match(/^Title:\s*(.+)$/m);
  const title = match?.[1]?.trim();
  return title && title.length > 0 ? title : "Untitled Play";
}

function slugify(value: string): string {
  return value
    .replace(/[^a-z0-9]+/gi, "-")
    .replace(/^-+|-+$/g, "")
    .toLowerCase() || "untitled";
}

function styleSlug(style: string): string {
  return style === "condensed" ? "acting-edition" : "manuscript";
}

export function setupPdfExport(
  button: HTMLButtonElement,
  styleSelect: HTMLSelectElement,
  getView: () => EditorView | null,
) {
  button.addEventListener("click", () => {
    const view = getView();
    if (!view) return;

    const source = view.state.doc.toString();
    const style = styleSelect.value;
    const pdfBytes = renderPDF(source, style);
    if (!pdfBytes || pdfBytes.length === 0) return;

    const blob = new Blob([pdfBytes], { type: "application/pdf" });
    const url = URL.createObjectURL(blob);
    const filename = `${slugify(extractDraftTitle(source))}-${styleSlug(style)}.pdf`;

    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    a.click();

    URL.revokeObjectURL(url);
  });
}
