import * as path from "node:path";

// ---------------------------------------------------------------------------
// Minimal interfaces that mirror the VS Code API surface used by the
// extracted functions. Real vscode.TextDocument / FoldingRange / etc.
// satisfy these structurally, so extension.ts can pass them straight through.
// ---------------------------------------------------------------------------

export interface TextLineLike {
	readonly text: string;
}

export interface TextDocumentLike {
	readonly lineCount: number;
	lineAt(line: number): TextLineLike;
	readonly uri: { readonly fsPath: string };
}

export interface PositionLike {
	readonly line: number;
	readonly character: number;
}

export interface RangeLike {
	readonly start: PositionLike;
	readonly end: PositionLike;
}

export interface DiagnosticLike {
	readonly range: RangeLike;
	readonly message: string;
	readonly severity: number;
	source?: string;
}

export interface FoldingRangeLike {
	readonly start: number;
	readonly end: number;
}

/** Constructors injected by the caller so lib stays VS Code-free. */
export interface VscodeFactories {
	Range: new (startLine: number, startChar: number, endLine: number, endChar: number) => RangeLike;
	Position: new (line: number, character: number) => PositionLike;
	Diagnostic: new (range: RangeLike, message: string, severity: number) => DiagnosticLike;
	DiagnosticSeverity: { readonly Error: number };
	FoldingRange: new (start: number, end: number) => FoldingRangeLike;
}

// ---------------------------------------------------------------------------
// Pure functions
// ---------------------------------------------------------------------------

export function replaceExtension(filePath: string, extension: string): string {
	return path.join(
		path.dirname(filePath),
		`${path.basename(filePath, path.extname(filePath))}${extension}`,
	);
}

export function validateServerPath(serverPath: string): string {
	serverPath = serverPath.trim();
	if (!serverPath) {
		throw new Error("downstage.server.path must not be empty");
	}
	if (/[\u0000-\u001f]/u.test(serverPath)) {
		throw new Error("downstage.server.path contains control characters");
	}
	return serverPath;
}

const allowedRenderStyles = new Set(["standard", "condensed"]);

export function getValidatedRenderStyle(style: string): string {
	if (!allowedRenderStyles.has(style)) {
		throw new Error(`Unsupported render style: ${style}`);
	}
	return style;
}

export function getPreviewHtml(body: string): string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<style>
html, body { margin: 0; padding: 0; height: 100%; }
body { overflow: hidden; }
.preview-container {
	position: relative;
	width: 100%;
	height: 100%;
}
.preview-frame {
	position: absolute;
	top: 0;
	left: 0;
	width: 100%;
	height: 100%;
	border: 0;
	visibility: hidden;
}
</style>
</head>
<body>
<div class="preview-container">
	<iframe id="preview-a" class="preview-frame" sandbox="allow-same-origin"></iframe>
	<iframe id="preview-b" class="preview-frame" sandbox="allow-same-origin"></iframe>
</div>
<script>
	const frameA = document.getElementById("preview-a");
	const frameB = document.getElementById("preview-b");
	let active = frameA;
	let staging = frameB;
	let pendingLine = null;
	let loadGeneration = 0;
	let clearTimer = null;

	function updatePreview(html, line) {
		if (typeof line === "number") {
			pendingLine = line;
		}
		if (clearTimer !== null) {
			clearTimeout(clearTimer);
			clearTimer = null;
		}
		loadGeneration++;
		staging.dataset.generation = String(loadGeneration);
		staging.srcdoc = html;
	}

	function scrollPreviewToLine(line, behavior = "smooth", frame) {
		const targetFrame = frame || active;
		const doc = targetFrame.contentDocument;
		if (!doc) {
			pendingLine = line;
			return;
		}

		const els = doc.querySelectorAll("[data-source-line]");
		let target = null;
		for (const el of els) {
			const sourceLine = parseInt(el.getAttribute("data-source-line"), 10);
			if (sourceLine <= line) {
				target = el;
			} else {
				break;
			}
		}
		if (target) {
			target.scrollIntoView({ behavior, block: "center" });
		}
	}

	function onFrameLoad(frame) {
		if (frame !== staging) return;
		if (frame.dataset.generation !== String(loadGeneration)) return;
		if (pendingLine !== null) {
			const line = pendingLine;
			pendingLine = null;
			scrollPreviewToLine(line, "auto", frame);
		}
		staging.style.visibility = "visible";
		active.style.visibility = "hidden";
		const retired = active;
		active = staging;
		staging = retired;
		retired.dataset.generation = "-1";
		clearTimer = setTimeout(() => {
			clearTimer = null;
			retired.srcdoc = "";
		}, 0);
	}

	frameA.addEventListener("load", () => onFrameLoad(frameA));
	frameB.addEventListener("load", () => onFrameLoad(frameB));

	window.addEventListener("message", (event) => {
		const msg = event.data;
		if (msg.type === "update") {
			updatePreview(msg.html, msg.line);
		}
		if (msg.type === "scrollTo") {
			scrollPreviewToLine(msg.line);
		}
	});

	updatePreview(${JSON.stringify(body)});
</script>
</body>
</html>`;
}

export function isCueSuggestionLine(document: TextDocumentLike, line: number): boolean {
	if (line <= 0 || line >= document.lineCount) {
		return false;
	}

	const currentLine = document.lineAt(line).text;
	if (currentLine.trim() !== "") {
		return false;
	}

	const previousLine = document.lineAt(line - 1).text;
	return previousLine.trim() === "";
}

export function parseRenderDiagnostics(
	document: TextDocumentLike,
	stderr: string,
	factories: VscodeFactories,
): DiagnosticLike[] {
	const diagnostics: DiagnosticLike[] = [];
	const lines = stderr.split(/\r?\n/);
	const fileName = path.basename(document.uri.fsPath);
	const pattern = /^([^:]+):(\d+):(\d+):\s+(.*)$/;

	for (const line of lines) {
		const match = pattern.exec(line);
		if (!match || match[1] !== fileName) {
			continue;
		}

		const lineNumber = Number(match[2]) - 1;
		const columnNumber = Number(match[3]) - 1;
		if (lineNumber < 0 || lineNumber >= document.lineCount) {
			continue;
		}

		const documentLine = document.lineAt(lineNumber);
		const character = Math.min(Math.max(columnNumber, 0), documentLine.text.length);
		const range = new factories.Range(
			lineNumber, character,
			lineNumber, documentLine.text.length,
		);
		const diagnostic = new factories.Diagnostic(
			range,
			match[4],
			factories.DiagnosticSeverity.Error,
		);
		diagnostic.source = "downstage-render";
		diagnostics.push(diagnostic);
	}

	return diagnostics;
}

export function provideDownstageFoldingRanges(
	document: TextDocumentLike,
	FoldingRange: new (start: number, end: number) => FoldingRangeLike,
): FoldingRangeLike[] {
	const ranges: FoldingRangeLike[] = [];
	const sectionStack: Array<{ level: number; line: number }> = [];
	let songStartLine: number | undefined;

	for (let line = 0; line < document.lineCount; line++) {
		const text = document.lineAt(line).text.trim();
		const heading = /^(#{1,3})\s+/.exec(text);
		if (heading) {
			const level = heading[1].length;
			while (sectionStack.length > 0 && sectionStack[sectionStack.length - 1].level >= level) {
				const section = sectionStack.pop();
				if (section && line - section.line > 1) {
					ranges.push(new FoldingRange(section.line, line - 1));
				}
			}
			sectionStack.push({ level, line });
			continue;
		}

		if (text === "SONG END" && songStartLine !== undefined && line > songStartLine) {
			ranges.push(new FoldingRange(songStartLine, line));
			songStartLine = undefined;
			continue;
		}

		if (text.startsWith("SONG")) {
			songStartLine = line;
		}
	}

	while (sectionStack.length > 0) {
		const section = sectionStack.pop();
		if (section && document.lineCount - section.line > 1) {
			ranges.push(new FoldingRange(section.line, document.lineCount - 1));
		}
	}

	return ranges;
}

export class DownstageRenderError extends Error {
	readonly stderr: string;

	constructor(message: string, stderr: string) {
		super(message);
		this.stderr = stderr;
	}
}
