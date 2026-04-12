import type { Text } from "@codemirror/state";

// Convert an LSP (line, character) position into a CodeMirror document
// offset. LSP characters are UTF-16 code units within the line, which
// matches how CodeMirror stores line content, so a direct add works.
export function offsetFromLSP(doc: Text, line: number, character: number): number {
  if (line < 0) return 0;
  if (line >= doc.lines) {
    const last = doc.line(doc.lines);
    return Math.min(last.to, last.from + Math.max(0, character));
  }
  const lineObj = doc.line(line + 1);
  const offset = lineObj.from + Math.max(0, character);
  return Math.min(offset, lineObj.to);
}
