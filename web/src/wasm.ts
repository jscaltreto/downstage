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
  const response = await fetch("downstage.wasm");
  if (!response.ok) {
    throw new Error(`failed to fetch downstage.wasm: ${response.status} ${response.statusText}`);
  }

  let result: WebAssembly.WebAssemblyInstantiatedSource;
  try {
    result = await WebAssembly.instantiateStreaming(response, go.importObject);
  } catch {
    // Some static hosts do not serve .wasm with the correct MIME type.
    const bytes = await response.arrayBuffer();
    result = await WebAssembly.instantiate(bytes, go.importObject);
  }

  const runPromise = go.run(result.instance);
  await waitForDownstage(runPromise);
}

async function waitForDownstage(runPromise: Promise<void>): Promise<void> {
  const timeoutMs = 5000;
  const start = performance.now();

  while (!window.downstage) {
    if (performance.now() - start >= timeoutMs) {
      throw new Error("timed out waiting for WASM runtime to initialize");
    }

    const state = await Promise.race([
      new Promise<"tick">((resolve) => setTimeout(() => resolve("tick"), 0)),
      runPromise.then(() => "exit" as const),
    ]);

    if (state === "exit" && !window.downstage) {
      throw new Error("WASM runtime exited before initialization completed");
    }
  }
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
