import * as path from "node:path";
import { describe, expect, it } from "vitest";
import {
	type DiagnosticLike,
	type FoldingRangeLike,
	type RangeLike,
	type TextDocumentLike,
	type VscodeFactories,
	DownstageRenderError,
	getPreviewHtml,
	getValidatedRenderStyle,
	isCueSuggestionLine,
	parseRenderDiagnostics,
	provideDownstageFoldingRanges,
	replaceExtension,
	validateServerPath,
} from "./lib";

// ---------------------------------------------------------------------------
// Helpers — lightweight stubs that satisfy the lib interfaces
// ---------------------------------------------------------------------------

function makeDocument(lines: string[], fsPath = "test.ds"): TextDocumentLike {
	return {
		lineCount: lines.length,
		lineAt(line: number) {
			return { text: lines[line] };
		},
		uri: { fsPath },
	};
}

class StubPosition {
	constructor(
		readonly line: number,
		readonly character: number,
	) {}
}

class StubRange implements RangeLike {
	readonly start: StubPosition;
	readonly end: StubPosition;
	constructor(startLine: number, startChar: number, endLine: number, endChar: number) {
		this.start = new StubPosition(startLine, startChar);
		this.end = new StubPosition(endLine, endChar);
	}
}

class StubDiagnostic implements DiagnosticLike {
	source?: string;
	constructor(
		readonly range: RangeLike,
		readonly message: string,
		readonly severity: number,
	) {}
}

class StubFoldingRange implements FoldingRangeLike {
	constructor(
		readonly start: number,
		readonly end: number,
	) {}
}

const factories: VscodeFactories = {
	Range: StubRange as unknown as VscodeFactories["Range"],
	Position: StubPosition as unknown as VscodeFactories["Position"],
	Diagnostic: StubDiagnostic as unknown as VscodeFactories["Diagnostic"],
	DiagnosticSeverity: { Error: 0 },
	FoldingRange: StubFoldingRange as unknown as VscodeFactories["FoldingRange"],
};

// ---------------------------------------------------------------------------
// replaceExtension
// ---------------------------------------------------------------------------

describe("replaceExtension", () => {
	it("swaps .ds for .pdf", () => {
		expect(replaceExtension(path.join("scripts", "play.ds"), ".pdf")).toBe(
			path.join("scripts", "play.pdf"),
		);
	});

	it("handles files with no extension", () => {
		expect(replaceExtension(path.join("scripts", "play"), ".pdf")).toBe(
			path.join("scripts", "play.pdf"),
		);
	});

	it("handles nested paths", () => {
		expect(replaceExtension(path.join("a", "b", "c", "play.ds"), ".html")).toBe(
			path.join("a", "b", "c", "play.html"),
		);
	});

	it("handles dotfiles", () => {
		expect(replaceExtension(path.join("scripts", ".hidden.ds"), ".pdf")).toBe(
			path.join("scripts", ".hidden.pdf"),
		);
	});
});

// ---------------------------------------------------------------------------
// validateServerPath
// ---------------------------------------------------------------------------

describe("validateServerPath", () => {
	it("returns a valid path unchanged", () => {
		expect(validateServerPath("downstage")).toBe("downstage");
	});

	it("trims whitespace", () => {
		expect(validateServerPath("  downstage  ")).toBe("downstage");
	});

	it("throws on empty string", () => {
		expect(() => validateServerPath("")).toThrow("must not be empty");
	});

	it("throws on whitespace-only string", () => {
		expect(() => validateServerPath("   ")).toThrow("must not be empty");
	});

	it("throws on control characters", () => {
		expect(() => validateServerPath("down\x00stage")).toThrow("control characters");
		expect(() => validateServerPath("down\nstage")).toThrow("control characters");
	});
});

// ---------------------------------------------------------------------------
// getValidatedRenderStyle
// ---------------------------------------------------------------------------

describe("getValidatedRenderStyle", () => {
	it("accepts standard", () => {
		expect(getValidatedRenderStyle("standard")).toBe("standard");
	});

	it("accepts condensed", () => {
		expect(getValidatedRenderStyle("condensed")).toBe("condensed");
	});

	it("rejects unknown styles", () => {
		expect(() => getValidatedRenderStyle("fancy")).toThrow("Unsupported render style");
	});
});

// ---------------------------------------------------------------------------
// getPreviewHtml
// ---------------------------------------------------------------------------

describe("getPreviewHtml", () => {
	it("returns a complete HTML document", () => {
		const html = getPreviewHtml("");
		expect(html).toContain("<!DOCTYPE html>");
		expect(html).toContain("</html>");
	});

	it("embeds the body in the updatePreview call", () => {
		const html = getPreviewHtml("<h1>Hello</h1>");
		expect(html).toContain(JSON.stringify("<h1>Hello</h1>"));
	});

	it("handles empty body", () => {
		const html = getPreviewHtml("");
		expect(html).toContain('updatePreview("")');
	});

	it("JSON-escapes special characters in body", () => {
		const html = getPreviewHtml('he said "hello"\nand left');
		expect(html).toContain(JSON.stringify('he said "hello"\nand left'));
	});

	it("includes the dual-iframe structure", () => {
		const html = getPreviewHtml("");
		expect(html).toContain('id="preview-a"');
		expect(html).toContain('id="preview-b"');
	});
});

// ---------------------------------------------------------------------------
// isCueSuggestionLine
// ---------------------------------------------------------------------------

describe("isCueSuggestionLine", () => {
	it("returns true for a blank line preceded by a blank line", () => {
		const doc = makeDocument(["ALICE", "", "", ""]);
		expect(isCueSuggestionLine(doc, 2)).toBe(true);
	});

	it("returns false when current line has content", () => {
		const doc = makeDocument(["ALICE", "", "BOB"]);
		expect(isCueSuggestionLine(doc, 2)).toBe(false);
	});

	it("returns false when previous line has content", () => {
		const doc = makeDocument(["ALICE", "Hello!", ""]);
		expect(isCueSuggestionLine(doc, 2)).toBe(false);
	});

	it("returns false at line 0", () => {
		const doc = makeDocument(["", "ALICE"]);
		expect(isCueSuggestionLine(doc, 0)).toBe(false);
	});

	it("returns false at lineCount (out of range)", () => {
		const doc = makeDocument(["ALICE", ""]);
		expect(isCueSuggestionLine(doc, 2)).toBe(false);
	});

	it("returns false for negative line", () => {
		const doc = makeDocument(["", ""]);
		expect(isCueSuggestionLine(doc, -1)).toBe(false);
	});
});

// ---------------------------------------------------------------------------
// parseRenderDiagnostics
// ---------------------------------------------------------------------------

describe("parseRenderDiagnostics", () => {
	it("parses a single diagnostic", () => {
		const doc = makeDocument(["# Act One", "ALICE", "Hello world"], "play.ds");
		const stderr = "play.ds:2:1: unexpected character cue";
		const diags = parseRenderDiagnostics(doc, stderr, factories);

		expect(diags).toHaveLength(1);
		expect(diags[0].message).toBe("unexpected character cue");
		expect(diags[0].range.start.line).toBe(1);
		expect(diags[0].range.start.character).toBe(0);
		expect(diags[0].source).toBe("downstage-render");
	});

	it("parses multiple diagnostics", () => {
		const doc = makeDocument(["line0", "line1", "line2"], "test.ds");
		const stderr = "test.ds:1:1: error one\ntest.ds:3:2: error two";
		const diags = parseRenderDiagnostics(doc, stderr, factories);

		expect(diags).toHaveLength(2);
		expect(diags[0].message).toBe("error one");
		expect(diags[1].message).toBe("error two");
	});

	it("ignores lines from other files", () => {
		const doc = makeDocument(["content"], "play.ds");
		const stderr = "other.ds:1:1: not for us";
		const diags = parseRenderDiagnostics(doc, stderr, factories);

		expect(diags).toHaveLength(0);
	});

	it("ignores non-matching lines", () => {
		const doc = makeDocument(["content"], "play.ds");
		const stderr = "some random error output\n";
		const diags = parseRenderDiagnostics(doc, stderr, factories);

		expect(diags).toHaveLength(0);
	});

	it("skips diagnostics with out-of-range line numbers", () => {
		const doc = makeDocument(["only line"], "play.ds");
		const stderr = "play.ds:99:1: way past end";
		const diags = parseRenderDiagnostics(doc, stderr, factories);

		expect(diags).toHaveLength(0);
	});

	it("clamps column to line length", () => {
		const doc = makeDocument(["short"], "play.ds");
		const stderr = "play.ds:1:999: past end of line";
		const diags = parseRenderDiagnostics(doc, stderr, factories);

		expect(diags).toHaveLength(1);
		expect(diags[0].range.start.character).toBe(5); // "short".length
	});

	it("handles Windows-style line endings in stderr", () => {
		const doc = makeDocument(["content"], "play.ds");
		const stderr = "play.ds:1:1: error\r\n";
		const diags = parseRenderDiagnostics(doc, stderr, factories);

		expect(diags).toHaveLength(1);
	});
});

// ---------------------------------------------------------------------------
// provideDownstageFoldingRanges
// ---------------------------------------------------------------------------

describe("provideDownstageFoldingRanges", () => {
	it("returns empty for an empty document", () => {
		const doc = makeDocument([]);
		const ranges = provideDownstageFoldingRanges(doc, StubFoldingRange);

		expect(ranges).toHaveLength(0);
	});

	it("folds a single heading section to end of document", () => {
		const doc = makeDocument([
			"# Act One",
			"ALICE",
			"Hello",
		]);
		const ranges = provideDownstageFoldingRanges(doc, StubFoldingRange);

		expect(ranges).toHaveLength(1);
		expect(ranges[0].start).toBe(0);
		expect(ranges[0].end).toBe(2);
	});

	it("folds nested headings", () => {
		const doc = makeDocument([
			"# Act One",
			"## Scene One",
			"ALICE",
			"Hello",
			"## Scene Two",
			"BOB",
			"Goodbye",
		]);
		const ranges = provideDownstageFoldingRanges(doc, StubFoldingRange);

		// ## Scene One: 1-3, ## Scene Two: 4-6, # Act One: 0-6
		expect(ranges).toHaveLength(3);
		const sorted = [...ranges].sort((a, b) => a.start - b.start);
		expect(sorted[0]).toEqual({ start: 0, end: 6 });
		expect(sorted[1]).toEqual({ start: 1, end: 3 });
		expect(sorted[2]).toEqual({ start: 4, end: 6 });
	});

	it("folds SONG/SONG END blocks", () => {
		const doc = makeDocument([
			"ALICE",
			"SONG My Song",
			"La la la",
			"SONG END",
			"BOB",
		]);
		const ranges = provideDownstageFoldingRanges(doc, StubFoldingRange);

		expect(ranges).toHaveLength(1);
		expect(ranges[0].start).toBe(1);
		expect(ranges[0].end).toBe(3);
	});

	it("handles mixed headings and songs", () => {
		const doc = makeDocument([
			"# Act One",
			"ALICE",
			"SONG Opener",
			"La la",
			"SONG END",
			"BOB",
		]);
		const ranges = provideDownstageFoldingRanges(doc, StubFoldingRange);

		const sorted = [...ranges].sort((a, b) => a.start - b.start);
		expect(sorted).toHaveLength(2);
		expect(sorted[0]).toEqual({ start: 0, end: 5 }); // # Act One
		expect(sorted[1]).toEqual({ start: 2, end: 4 }); // SONG
	});

	it("does not fold single-line sections", () => {
		const doc = makeDocument([
			"# Act One",
			"# Act Two",
		]);
		const ranges = provideDownstageFoldingRanges(doc, StubFoldingRange);

		// # Act One is only 1 line (0-0), so not folded. # Act Two: 1 to end (1) = single line, not folded.
		expect(ranges).toHaveLength(0);
	});

	it("folds unclosed sections to end of document", () => {
		const doc = makeDocument([
			"# Act One",
			"## Scene One",
			"ALICE",
			"Hello",
			"Goodbye",
		]);
		const ranges = provideDownstageFoldingRanges(doc, StubFoldingRange);

		const sorted = [...ranges].sort((a, b) => a.start - b.start);
		expect(sorted).toHaveLength(2);
		expect(sorted[0]).toEqual({ start: 0, end: 4 });
		expect(sorted[1]).toEqual({ start: 1, end: 4 });
	});
});

// ---------------------------------------------------------------------------
// DownstageRenderError
// ---------------------------------------------------------------------------

describe("DownstageRenderError", () => {
	it("stores stderr and message", () => {
		const err = new DownstageRenderError("render failed", "some stderr output");
		expect(err.message).toBe("render failed");
		expect(err.stderr).toBe("some stderr output");
		expect(err).toBeInstanceOf(Error);
	});
});
