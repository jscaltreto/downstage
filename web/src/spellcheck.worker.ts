/// <reference lib="webworker" />

import {
  checkWords,
  loadSpellDictionary,
  suggestForWord,
  type SpellDictionary,
} from "./spellcheck-engine";

type Request =
  | { type: "check"; id: number; words: string[] }
  | { type: "suggest"; id: number; word: string };

type Response =
  | { type: "ready" }
  | { type: "init-error"; message: string }
  | { type: "check"; id: number; results: Record<string, boolean> }
  | { type: "suggest"; id: number; suggestions: string[] };

const scope = self as unknown as DedicatedWorkerGlobalScope;

let dict: SpellDictionary | null = null;

loadSpellDictionary()
  .then((loaded) => {
    dict = loaded;
    scope.postMessage({ type: "ready" } satisfies Response);
  })
  .catch((err: unknown) => {
    const message = err instanceof Error ? err.message : String(err);
    scope.postMessage({ type: "init-error", message } satisfies Response);
  });

scope.addEventListener("message", (event: MessageEvent<Request>) => {
  const msg = event.data;
  if (!dict) return;
  if (msg.type === "check") {
    scope.postMessage({
      type: "check",
      id: msg.id,
      results: checkWords(dict, msg.words),
    } satisfies Response);
  } else if (msg.type === "suggest") {
    scope.postMessage({
      type: "suggest",
      id: msg.id,
      suggestions: suggestForWord(dict, msg.word),
    } satisfies Response);
  }
});
