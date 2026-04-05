declare class Go {
  importObject: WebAssembly.Imports;
  run(instance: WebAssembly.Instance): Promise<void>;
}

declare global {
  interface Window {
    Go: typeof Go;
    downstage: {
      parse(source: string): { errors: ParseError[] };
      renderHTML(source: string, style?: string): string;
      renderPDF(source: string, style?: string): Uint8Array;
      semanticTokens(source: string): Uint32Array;
      tokenTypeNames: string[];
    };
  }
}

export interface ParseError {
  message: string;
  line: number;
  col: number;
  endLine: number;
  endCol: number;
}

export async function initWasm(): Promise<void> {
  const go = new window.Go();
  const result = await WebAssembly.instantiateStreaming(
    fetch("dist/downstage.wasm"),
    go.importObject,
  );
  go.run(result.instance);
}

export function parse(source: string) {
  return window.downstage.parse(source);
}

export function renderHTML(source: string, style?: string): string {
  return window.downstage.renderHTML(source, style);
}

export function renderPDF(source: string, style?: string): Uint8Array {
  return window.downstage.renderPDF(source, style);
}

export function semanticTokens(source: string): Uint32Array {
  return window.downstage.semanticTokens(source);
}

export function tokenTypeNames(): string[] {
  return Array.from(window.downstage.tokenTypeNames);
}
