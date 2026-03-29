import * as vscode from "vscode";
import { spawn } from "node:child_process";
import * as path from "node:path";
import {
	LanguageClient,
	LanguageClientOptions,
	ServerOptions,
	Trace,
} from "vscode-languageclient/node";

// Configuration section and setting keys
const configSection = "downstage";
const settingServerPath = "server.path";
const settingServerTrace = "server.trace";
const settingAutoSuggestCues = "editor.autoSuggestCharacterCues";
const settingRenderStyle = "render.style";
const settingOpenAfterRender = "render.openAfterRender";
const settingPreviewDebounce = "preview.debounceMs";

let client: LanguageClient | undefined;
const cueSuggestTimers = new Map<string, NodeJS.Timeout>();
let renderOutputChannel: vscode.OutputChannel | undefined;
let renderDiagnostics: vscode.DiagnosticCollection | undefined;
let extensionContext: vscode.ExtensionContext | undefined;
const allowedRenderStyles = new Set(["standard", "condensed"]);
const defaultServerPath = "downstage";
const trustedServerPathsKey = "downstage.trustedServerPaths";

interface PreviewState {
	panel: vscode.WebviewPanel;
	pending: ReturnType<typeof setTimeout> | undefined;
	child: ReturnType<typeof spawn> | undefined;
	lastHtml: string;
	requestId: number;
}
const previewPanels = new Map<string, PreviewState>();

export async function activate(context: vscode.ExtensionContext): Promise<void> {
	extensionContext = context;
	renderDiagnostics = vscode.languages.createDiagnosticCollection("downstage-render");

	const restartCommand = vscode.commands.registerCommand(
		"downstage.restartLanguageServer",
		async () => {
			await restartLanguageServer(context);
		},
	);
	const renderCommand = vscode.commands.registerCommand(
		"downstage.renderCurrentScript",
		async () => {
			await renderCurrentScript("standard");
		},
	);
	const renderCompactCommand = vscode.commands.registerCommand(
		"downstage.renderCompactScript",
		async () => {
			await renderCurrentScript("condensed");
		},
	);
	const previewCommand = vscode.commands.registerCommand(
		"downstage.previewCurrentScript",
		async () => {
			await renderCurrentScript("standard", "internal");
		},
	);
	const previewCompactCommand = vscode.commands.registerCommand(
		"downstage.previewCompactScript",
		async () => {
			await renderCurrentScript("condensed", "internal");
		},
	);
	const livePreviewCommand = vscode.commands.registerCommand(
		"downstage.livePreview",
		() => {
			openLivePreview();
		},
	);
	const foldingProvider = vscode.languages.registerFoldingRangeProvider(
		{ language: "downstage" },
		{
			provideFoldingRanges(document) {
				return provideDownstageFoldingRanges(document);
			},
		},
	);

	context.subscriptions.push(restartCommand);
	context.subscriptions.push(renderCommand);
	context.subscriptions.push(renderCompactCommand);
	context.subscriptions.push(previewCommand);
	context.subscriptions.push(previewCompactCommand);
	context.subscriptions.push(livePreviewCommand);
	context.subscriptions.push(foldingProvider);
	context.subscriptions.push(renderDiagnostics);
	context.subscriptions.push(
		vscode.workspace.onDidChangeConfiguration(async (event) => {
			if (!event.affectsConfiguration(`${configSection}.${settingServerPath}`) &&
				!event.affectsConfiguration(`${configSection}.${settingServerTrace}`)) {
				return;
			}

			await restartLanguageServer(context);
		}),
	);
	context.subscriptions.push(
		vscode.workspace.onDidChangeTextDocument((event) => {
			scheduleCueSuggestForDocument(event.document);
			schedulePreviewUpdate(event.document);
		}),
	);
	context.subscriptions.push(
		vscode.window.onDidChangeTextEditorSelection((event) => {
			scheduleCueSuggestForEditor(event.textEditor);
		}),
	);
	context.subscriptions.push(
		vscode.window.onDidChangeTextEditorSelection((event) => {
			if (previewPanels.has(event.textEditor.document.uri.toString())) {
				syncPreviewScroll(event.textEditor);
			}
		}),
	);
	context.subscriptions.push(
		vscode.workspace.onDidCloseTextDocument((document) => {
			const key = document.uri.toString();
			const state = previewPanels.get(key);
			if (state) {
				state.panel.dispose();
			}
		}),
	);
	await startLanguageServer(context);
}

export async function deactivate(): Promise<void> {
	if (!client) {
		return;
	}

	await client.stop();
	client = undefined;
}

async function restartLanguageServer(context: vscode.ExtensionContext): Promise<void> {
	if (client) {
		await client.stop();
		client = undefined;
	}

	await startLanguageServer(context);
}

async function startLanguageServer(context: vscode.ExtensionContext): Promise<void> {
	const outputChannel = vscode.window.createOutputChannel("Downstage Language Server");
	context.subscriptions.push(outputChannel);
	const configuredServerPath = getServerPath();

	try {
		const serverPath = await getTrustedServerPath(configuredServerPath);
		const serverOptions: ServerOptions = {
			command: serverPath,
			args: ["lsp"],
			options: {
				cwd: getWorkspaceRoot(),
			},
		};

		const clientOptions: LanguageClientOptions = {
			documentSelector: [{ scheme: "file", language: "downstage" }],
			outputChannel,
		};

		client = new LanguageClient(
			"downstageLanguageServer",
			"Downstage Language Server",
			serverOptions,
			clientOptions,
		);

		client.setTrace(toTrace(getTraceSetting()));
		await client.start();
	} catch (error) {
		client = undefined;
		const message = [
			"Failed to start the Downstage language server.",
			`Expected executable: ${configuredServerPath}`,
			"Install the `downstage` binary or set `downstage.server.path`.",
		].join(" ");
		outputChannel.appendLine(String(error));
		void vscode.window.showErrorMessage(message, "Open Settings").then((selection) => {
			if (selection === "Open Settings") {
				void vscode.commands.executeCommand(
					"workbench.action.openSettings",
					`${configSection}.${settingServerPath}`,
				);
			}
		});
	}
}

function getServerPath(): string {
	return vscode.workspace.getConfiguration(configSection).get<string>(settingServerPath, defaultServerPath);
}

function getValidatedServerPath(): string {
	return validateServerPath(getServerPath());
}

function validateServerPath(serverPath: string): string {
	serverPath = serverPath.trim();
	if (!serverPath) {
		throw new Error("downstage.server.path must not be empty");
	}
	if (/[\u0000-\u001f]/u.test(serverPath)) {
		throw new Error("downstage.server.path contains control characters");
	}
	return serverPath;
}

async function getTrustedServerPath(configuredPath?: string): Promise<string> {
	const serverPath = validateServerPath(configuredPath ?? getServerPath());
	if (serverPath === defaultServerPath || isTrustedServerPath(serverPath)) {
		return serverPath;
	}

	const selection = await vscode.window.showWarningMessage(
		`Downstage is configured to launch a custom executable: ${serverPath}`,
		{ modal: true },
		"Trust and Launch",
	);
	if (selection !== "Trust and Launch") {
		throw new Error("custom downstage.server.path was not trusted");
	}

	await rememberTrustedServerPath(serverPath);
	return serverPath;
}

function isTrustedServerPath(serverPath: string): boolean {
	return getTrustedServerPaths().includes(serverPath);
}

function getTrustedServerPaths(): string[] {
	return extensionContext?.workspaceState.get<string[]>(trustedServerPathsKey, []) ?? [];
}

async function rememberTrustedServerPath(serverPath: string): Promise<void> {
	if (!extensionContext) {
		return;
	}

	const trustedPaths = new Set(getTrustedServerPaths());
	trustedPaths.add(serverPath);
	await extensionContext.workspaceState.update(trustedServerPathsKey, Array.from(trustedPaths));
}

function getTraceSetting(): string {
	return vscode.workspace.getConfiguration(configSection).get<string>(settingServerTrace, "off");
}

function getAutoSuggestSetting(): boolean {
	return vscode.workspace.getConfiguration(configSection).get<boolean>(
		settingAutoSuggestCues,
		true,
	);
}

function getRenderStyleSetting(): string {
	return vscode.workspace.getConfiguration(configSection).get<string>(
		settingRenderStyle,
		"standard",
	);
}

function getOpenAfterRenderSetting(): boolean {
	return vscode.workspace.getConfiguration(configSection).get<boolean>(
		settingOpenAfterRender,
		true,
	);
}

function toTrace(value: string): Trace {
	switch (value) {
		case "messages":
			return Trace.Messages;
		case "verbose":
			return Trace.Verbose;
		default:
			return Trace.Off;
	}
}

function getWorkspaceRoot(): string | undefined {
	const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
	if (!workspaceFolder) {
		return undefined;
	}

	return workspaceFolder.uri.fsPath;
}

function provideDownstageFoldingRanges(document: vscode.TextDocument): vscode.FoldingRange[] {
	const ranges: vscode.FoldingRange[] = [];
	const sectionStack: Array<{ level: number; line: number }> = [];
	let songStartLine: number | undefined;

	for (let line = 0; line < document.lineCount; line++) {
		const text = document.lineAt(line).text.trim();
		const heading = /^(#{1,3})\s+/.exec(text);
		if (heading) {
			const level = heading[1].length;
			while (sectionStack.length > 0 && sectionStack[sectionStack.length - 1].level >= level) {
				const section = sectionStack.pop();
				if (section && line-section.line > 1) {
					ranges.push(new vscode.FoldingRange(section.line, line - 1));
				}
			}
			sectionStack.push({ level, line });
			continue;
		}

		if (text === "SONG END" && songStartLine !== undefined && line > songStartLine) {
			ranges.push(new vscode.FoldingRange(songStartLine, line));
			songStartLine = undefined;
			continue;
		}

		if (text.startsWith("SONG")) {
			songStartLine = line;
		}
	}

	while (sectionStack.length > 0) {
		const section = sectionStack.pop();
		if (section && document.lineCount-section.line > 1) {
			ranges.push(new vscode.FoldingRange(section.line, document.lineCount - 1));
		}
	}

	return ranges;
}

function scheduleCueSuggestForDocument(document: vscode.TextDocument): void {
	const editor = vscode.window.activeTextEditor;
	if (!editor || editor.document.uri.toString() !== document.uri.toString()) {
		return;
	}

	scheduleCueSuggestForEditor(editor);
}

function scheduleCueSuggestForEditor(editor: vscode.TextEditor): void {
	if (!getAutoSuggestSetting()) {
		return;
	}

	if (editor.document.languageId !== "downstage") {
		return;
	}

	if (!editor.selection.isEmpty || editor.selections.length !== 1) {
		return;
	}

	const key = editor.document.uri.toString();
	const existing = cueSuggestTimers.get(key);
	if (existing) {
		clearTimeout(existing);
	}

	const timer = setTimeout(() => {
		cueSuggestTimers.delete(key);
		void maybeTriggerCueSuggest(editor);
	}, 0);
	cueSuggestTimers.set(key, timer);
}

async function maybeTriggerCueSuggest(editor: vscode.TextEditor): Promise<void> {
	if (vscode.window.activeTextEditor !== editor) {
		return;
	}

	const line = editor.selection.active.line;
	if (!isCueSuggestionLine(editor.document, line)) {
		return;
	}

	await vscode.commands.executeCommand("editor.action.triggerSuggest");
}

function isCueSuggestionLine(document: vscode.TextDocument, line: number): boolean {
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

type RenderOpenMode = "config" | "internal";

async function renderCurrentScript(
	styleOverride?: string,
	openMode: RenderOpenMode = "config",
): Promise<void> {
	const editor = vscode.window.activeTextEditor;
	if (!editor || editor.document.languageId !== "downstage") {
		void vscode.window.showErrorMessage("Open a Downstage script before rendering.");
		return;
	}

	if (editor.document.isUntitled) {
		void vscode.window.showErrorMessage("Save the script before rendering.");
		return;
	}

	await editor.document.save();

	const inputPath = editor.document.uri.fsPath;
	const outputPath = replaceExtension(inputPath, ".pdf");
	const outputChannel = getRenderOutputChannel();

	outputChannel.clear();

	try {
		const serverPath = await getTrustedServerPath();
		const style = getValidatedRenderStyle(styleOverride ?? getRenderStyleSetting());
		outputChannel.appendLine(`Running: ${serverPath} render --style ${style} ${inputPath}`);
		outputChannel.show(true);
		renderDiagnostics?.delete(editor.document.uri);

		await runDownstageRender(serverPath, style, inputPath, outputChannel);
		renderDiagnostics?.delete(editor.document.uri);
		const message = `Rendered PDF: ${path.basename(outputPath)}`;
		if (openMode === "internal") {
			await openRenderedPdf(vscode.Uri.file(outputPath));
			void vscode.window.showInformationMessage(message);
			return;
		}

		if (!getOpenAfterRenderSetting()) {
			void vscode.window.showInformationMessage(message);
			return;
		}

		const selection = await vscode.window.showInformationMessage(message, "Open PDF");
		if (selection === "Open PDF") {
			await vscode.env.openExternal(vscode.Uri.file(outputPath));
		}
	} catch (error) {
		if (error instanceof DownstageRenderError) {
			renderDiagnostics?.set(
				editor.document.uri,
				parseRenderDiagnostics(editor.document, error.stderr),
			);
		}
		outputChannel.appendLine(String(error));
		outputChannel.show(true);
		void vscode.window.showErrorMessage(
			"Downstage render failed. See 'Downstage Render' output for details.",
		);
	}
}

function getRenderOutputChannel(): vscode.OutputChannel {
	if (!renderOutputChannel) {
		renderOutputChannel = vscode.window.createOutputChannel("Downstage Render");
	}
	return renderOutputChannel;
}

function getValidatedRenderStyle(style: string): string {
	if (!allowedRenderStyles.has(style)) {
		throw new Error(`Unsupported render style: ${style}`);
	}
	return style;
}

async function runDownstageRender(
	serverPath: string,
	style: string,
	inputPath: string,
	outputChannel: vscode.OutputChannel,
): Promise<void> {
	await new Promise<void>((resolve, reject) => {
		let stderr = "";
		const child = spawn(serverPath, ["render", "--style", style, inputPath], {
			cwd: path.dirname(inputPath),
		});

		const timeout = setTimeout(() => {
			child.kill();
			reject(new Error("downstage render timed out after 60 seconds"));
		}, 60_000);

		child.stdout.on("data", (chunk: Buffer | string) => {
			outputChannel.append(chunk.toString());
		});
		child.stderr.on("data", (chunk: Buffer | string) => {
			const text = chunk.toString();
			stderr += text;
			outputChannel.append(text);
		});
		child.on("error", (error) => {
			clearTimeout(timeout);
			reject(error);
		});
		child.on("close", (code) => {
			clearTimeout(timeout);
			if (code === 0) {
				resolve();
				return;
			}
			reject(new DownstageRenderError(
				`downstage render exited with code ${code ?? "unknown"}`,
				stderr,
			));
		});
	});
}

async function openRenderedPdf(uri: vscode.Uri): Promise<void> {
	try {
		await vscode.commands.executeCommand("vscode.open", uri);
	} catch {
		await vscode.env.openExternal(uri);
	}
}

function parseRenderDiagnostics(
	document: vscode.TextDocument,
	stderr: string,
): vscode.Diagnostic[] {
	const diagnostics: vscode.Diagnostic[] = [];
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
		const range = new vscode.Range(
			new vscode.Position(lineNumber, character),
			new vscode.Position(lineNumber, documentLine.text.length),
		);
		const diagnostic = new vscode.Diagnostic(
			range,
			match[4],
			vscode.DiagnosticSeverity.Error,
		);
		diagnostic.source = "downstage-render";
		diagnostics.push(diagnostic);
	}

	return diagnostics;
}

function replaceExtension(filePath: string, extension: string): string {
	return path.join(
		path.dirname(filePath),
		`${path.basename(filePath, path.extname(filePath))}${extension}`,
	);
}

function getPreviewDebounceSetting(): number {
	return vscode.workspace.getConfiguration(configSection).get<number>(
		settingPreviewDebounce,
		300,
	);
}

function openLivePreview(): void {
	const editor = vscode.window.activeTextEditor;
	if (!editor || editor.document.languageId !== "downstage") {
		void vscode.window.showErrorMessage("Open a Downstage script to preview.");
		return;
	}

	const key = editor.document.uri.toString();
	const existing = previewPanels.get(key);
	if (existing) {
		existing.panel.reveal(vscode.ViewColumn.Beside);
		return;
	}

	const panel = vscode.window.createWebviewPanel(
		"downstagePreview",
		`Preview: ${path.basename(editor.document.uri.fsPath)}`,
		vscode.ViewColumn.Beside,
		{ enableScripts: true, retainContextWhenHidden: true },
	);

	const state: PreviewState = {
		panel,
		pending: undefined,
		child: undefined,
		lastHtml: "",
		requestId: 0,
	};

	panel.webview.html = getPreviewHtml("");
	panel.onDidDispose(() => {
		if (state.pending) {
			clearTimeout(state.pending);
		}
		if (state.child) {
			state.child.kill();
		}
		previewPanels.delete(key);
	});

	previewPanels.set(key, state);
	void renderToPreview(editor.document, state);
}

function getPreviewHtml(body: string): string {
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

function schedulePreviewUpdate(document: vscode.TextDocument): void {
	const key = document.uri.toString();
	const state = previewPanels.get(key);
	if (!state) {
		return;
	}

	if (state.pending) {
		clearTimeout(state.pending);
	}

	state.pending = setTimeout(() => {
		state.pending = undefined;
		void renderToPreview(document, state);
	}, getPreviewDebounceSetting());
}

async function renderToPreview(document: vscode.TextDocument, state: PreviewState): Promise<void> {
	if (state.child) {
		state.child.kill();
		state.child = undefined;
	}

	state.requestId++
	const requestId = state.requestId

	let serverPath: string;
	let style: string;
	try {
		serverPath = await getTrustedServerPath();
		style = getValidatedRenderStyle(getRenderStyleSetting());
	} catch (error) {
		const outputChannel = getRenderOutputChannel();
		outputChannel.appendLine(`Preview render error: ${String(error)}`);
		return;
	}
	if (requestId !== state.requestId) {
		return;
	}
	const sourceName = path.basename(document.uri.fsPath);
	const child = spawn(serverPath, [
		"render", "--stdin", "--stdout",
		"--format", "html",
		"--source-anchors",
		"--style", style,
		"--source-name", sourceName,
	], {
		cwd: path.dirname(document.uri.fsPath),
	});

	state.child = child;

	const stdoutChunks: string[] = [];
	const stderrChunks: string[] = [];

	child.stdout.on("data", (chunk: Buffer | string) => {
		stdoutChunks.push(chunk.toString());
	});
	child.stderr.on("data", (chunk: Buffer | string) => {
		stderrChunks.push(chunk.toString());
	});

	child.on("error", (err) => {
		if (state.child !== child) {
			return;
		}
		state.child = undefined;
		const outputChannel = getRenderOutputChannel();
		outputChannel.appendLine(`Preview render error: ${err.message}`);
	});

	child.on("close", (code) => {
		if (state.child !== child) {
			return;
		}
		state.child = undefined;
		const stdout = stdoutChunks.join("");
		const stderr = stderrChunks.join("");
		if (code === 0) {
			state.lastHtml = stdout;
			renderDiagnostics?.delete(document.uri);
			const editor = vscode.window.activeTextEditor;
			const line = editor && editor.document.uri.toString() === document.uri.toString()
				? editor.selection.active.line + 1
				: undefined;
			void state.panel.webview.postMessage({ type: "update", html: stdout, line });
		} else {
			if (stderr) {
				renderDiagnostics?.set(
					document.uri,
					parseRenderDiagnostics(document, stderr),
				);
				const outputChannel = getRenderOutputChannel();
				outputChannel.appendLine(stderr);
			}
			if (state.lastHtml) {
				void state.panel.webview.postMessage({ type: "update", html: state.lastHtml });
			}
		}
	});

	child.stdin.on("error", () => {
		// Process may have died before we finished writing - ignore
	});
	child.stdin.write(document.getText());
	child.stdin.end();
}

function syncPreviewScroll(editor: vscode.TextEditor): void {
	const key = editor.document.uri.toString();
	const state = previewPanels.get(key);
	if (!state) {
		return;
	}

	const line = editor.selection.active.line + 1; // 1-based to match data-source-line
	void state.panel.webview.postMessage({ type: "scrollTo", line });
}

class DownstageRenderError extends Error {
	readonly stderr: string;

	constructor(message: string, stderr: string) {
		super(message);
		this.stderr = stderr;
	}
}
