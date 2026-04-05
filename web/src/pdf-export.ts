import type { EditorView } from "@codemirror/view";
import { renderPDF } from "./wasm";

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

    const a = document.createElement("a");
    a.href = url;
    a.download = "manuscript.pdf";
    a.click();

    URL.revokeObjectURL(url);
  });
}
