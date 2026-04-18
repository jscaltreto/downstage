import * as vscode from "vscode";
import { spawn } from "node:child_process";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import {
	LanguageClient,
	LanguageClientOptions,
	ServerOptions,
	Trace,
} from "vscode-languageclient/node";
import {
	buildPDFPreviewArgs,
	buildPDFRenderArgs,
	type VscodeFactories,
	DownstageRenderError,
	findTitleValueSelection,
	getNewPlayTemplate,
	getPageSizeDisplayName,
	getPreviewHtml,
	getRenderStyleDisplayName,
	getSamplePlayTemplate,
	getValidatedPageSize,
	getValidatedRenderStyle,
	isCueSuggestionLine,
	parseRenderDiagnostics,
	provideDownstageFoldingRanges,
	replaceExtension,
	validateServerPath,
} from "./lib";

// Cast needed: VscodeFactories uses minimal interfaces (RangeLike etc.)
// while vscode.Range/Diagnostic have richer signatures. Structurally
// compatible at runtime — the lib functions only use the minimal surface.
const vscodeFactories = {
	Range: vscode.Range,
	Position: vscode.Position,
	Diagnostic: vscode.Diagnostic,
	DiagnosticSeverity: vscode.DiagnosticSeverity,
	FoldingRange: vscode.FoldingRange,
} as unknown as VscodeFactories;

const configSection = "downstage";
const settingServerPath = "server.path";
const settingServerTrace = "server.trace";
const settingAutoSuggestCues = "editor.autoSuggestCharacterCues";
const settingRenderStyle = "render.style";
const settingRenderPageSize = "render.pageSize";
const settingOpenAfterRender = "render.openAfterRender";
const settingPreviewDebounce = "preview.debounceMs";

let client: LanguageClient | undefined;
const cueSuggestTimers = new Map<string, NodeJS.Timeout>();
let renderOutputChannel: vscode.OutputChannel | undefined;
let renderDiagnostics: vscode.DiagnosticCollection | undefined;
let extensionContext: vscode.ExtensionContext | undefined;
const allowedRenderStyles = new Set(["standard", "condensed"]);
const pathServerCommand = "downstage";
const trustedServerPathsKey = "downstage.trustedServerPaths";
const welcomeShownKey = "downstage.welcomeShown";
const helpUrl = "https://www.getdownstage.com/syntax/";
const bundledServerTargets = new Map<string, string>([
	["linux:x64", "linux-x64"],
	["darwin:x64", "darwin-x64"],
	["darwin:arm64", "darwin-arm64"],
	["win32:x64", "win32-x64"],
]);

interface ResolvedServerCommand {
	command: string;
	expectedLocation: string;
}

interface PreviewState {
	panel: vscode.WebviewPanel;
	pending: ReturnType<typeof setTimeout> | undefined;
	child: ReturnType<typeof spawn> | undefined;
	lastHtml: string;
	requestId: number;
}
const previewPanels = new Map<string, PreviewState>();
const previewTempFiles = new Set<string>();
let welcomePanel: vscode.WebviewPanel | undefined;

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
	const renderCondensedCommand = vscode.commands.registerCommand(
		"downstage.renderCondensedScript",
		async () => {
			await renderCurrentScript("condensed");
		},
	);
	const previewCommand = vscode.commands.registerCommand(
		"downstage.previewCurrentScript",
		async () => {
			await previewCurrentScriptPdf("standard");
		},
	);
	const previewCondensedCommand = vscode.commands.registerCommand(
		"downstage.previewCondensedScript",
		async () => {
			await previewCurrentScriptPdf("condensed");
		},
	);
	const livePreviewCommand = vscode.commands.registerCommand(
		"downstage.livePreview",
		async () => {
			await openLivePreview();
		},
	);
	const newPlayCommand = vscode.commands.registerCommand(
		"downstage.newPlay",
		async () => {
			await openTemplateDocument(getNewPlayTemplate());
		},
	);
	const openSamplePlayCommand = vscode.commands.registerCommand(
		"downstage.openSamplePlay",
		async () => {
			await openTemplateDocument(getSamplePlayTemplate());
		},
	);
	const openWelcomeCommand = vscode.commands.registerCommand(
		"downstage.openWelcome",
		() => {
			openWelcomePanel();
		},
	);
	const openHelpCommand = vscode.commands.registerCommand(
		"downstage.openHelp",
		async () => {
			await vscode.env.openExternal(vscode.Uri.parse(helpUrl));
		},
	);
	const foldingProvider = vscode.languages.registerFoldingRangeProvider(
		{ language: "downstage" },
		{
			provideFoldingRanges(document) {
				return provideDownstageFoldingRanges(document, vscode.FoldingRange);
			},
		},
	);

	context.subscriptions.push(restartCommand);
	context.subscriptions.push(renderCommand);
	context.subscriptions.push(renderCondensedCommand);
	context.subscriptions.push(previewCommand);
	context.subscriptions.push(previewCondensedCommand);
	context.subscriptions.push(livePreviewCommand);
	context.subscriptions.push(newPlayCommand);
	context.subscriptions.push(openSamplePlayCommand);
	context.subscriptions.push(openWelcomeCommand);
	context.subscriptions.push(openHelpCommand);
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
	await maybeShowWelcomeOnFirstRun(context);
}

export async function deactivate(): Promise<void> {
	for (const tempFile of previewTempFiles) {
		try {
			fs.unlinkSync(tempFile);
		} catch {
			// File may already be gone
		}
	}
	previewTempFiles.clear();

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

	try {
		const serverPath = await resolveServerCommand();
		const serverOptions: ServerOptions = {
			command: serverPath.command,
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
			"Downstage could not start yet.",
			"Open help for setup steps, or open settings to choose a Downstage app.",
		].join(" ");
		outputChannel.appendLine(String(error));
		void vscode.window.showErrorMessage(message, "Open Help", "Open Settings").then((selection) => {
			if (selection === "Open Help") {
				void vscode.commands.executeCommand("downstage.openHelp");
				return;
			}
			if (selection === "Open Settings") {
				void vscode.commands.executeCommand(
					"workbench.action.openSettings",
					`${configSection}.${settingServerPath}`,
				);
			}
		});
	}
}

async function maybeShowWelcomeOnFirstRun(context: vscode.ExtensionContext): Promise<void> {
	if (context.workspaceState.get<boolean>(welcomeShownKey, false)) {
		return;
	}

	await context.workspaceState.update(welcomeShownKey, true);
	openWelcomePanel();
}

function openWelcomePanel(): void {
	if (welcomePanel) {
		welcomePanel.reveal(vscode.ViewColumn.One);
		return;
	}

	welcomePanel = vscode.window.createWebviewPanel(
		"downstageWelcome",
		"Downstage: Start Writing",
		vscode.ViewColumn.One,
		{ enableCommandUris: true },
	);
	welcomePanel.webview.html = getWelcomeHtml();
	welcomePanel.onDidDispose(() => {
		welcomePanel = undefined;
	});
}

function getWelcomeHtml(): string {
	const newPlayCommandUri = vscode.Uri.parse("command:downstage.newPlay");
	const samplePlayCommandUri = vscode.Uri.parse("command:downstage.openSamplePlay");
	const livePreviewCommandUri = vscode.Uri.parse("command:downstage.livePreview");
	const helpCommandUri = vscode.Uri.parse("command:downstage.openHelp");

	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
:root {
	color-scheme: light dark;
	font-family: Georgia, "Times New Roman", serif;
}
body {
	margin: 0;
	padding: 32px 24px;
	background:
		radial-gradient(circle at top, rgba(211, 157, 70, 0.2), transparent 45%),
		linear-gradient(180deg, #151515 0%, #231d17 100%);
	color: #f8f1e8;
}
main {
	max-width: 720px;
	margin: 0 auto;
}
h1 {
	font-size: 2.4rem;
	line-height: 1.1;
	margin: 0 0 12px;
}
p {
	font-size: 1.05rem;
	line-height: 1.6;
	color: rgba(248, 241, 232, 0.88);
}
.actions {
	display: flex;
	flex-wrap: wrap;
	gap: 12px;
	margin: 28px 0 18px;
}
.action {
	display: inline-block;
	padding: 12px 18px;
	border-radius: 999px;
	text-decoration: none;
	font-weight: 600;
}
.action-primary {
	background: #f4c98a;
	color: #1d1408;
}
.action-secondary {
	border: 1px solid rgba(248, 241, 232, 0.25);
	color: #f8f1e8;
}
.note {
	padding: 18px;
	border-radius: 18px;
	background: rgba(255, 255, 255, 0.06);
}
</style>
</head>
<body>
<main>
<h1>Write a play. See the pages. Export the PDF.</h1>
<p>Downstage keeps the writing plain and the manuscript clean. Start with a blank play, explore a sample, or open live preview while you write.</p>
<div class="actions">
<a class="action action-primary" href="${newPlayCommandUri}">New Play</a>
<a class="action action-secondary" href="${samplePlayCommandUri}">Open Sample Play</a>
<a class="action action-secondary" href="${livePreviewCommandUri}">Open Live Preview</a>
</div>
<div class="note">
<p>If Downstage cannot start, open the help guide for setup steps or point the extension at a local Downstage app in settings.</p>
<p><a class="action action-secondary" href="${helpCommandUri}">Open Help</a></p>
</div>
</main>
</body>
</html>`;
}

function getExplicitServerPath(): string | undefined {
	const inspection = vscode.workspace.getConfiguration(configSection).inspect<string>(settingServerPath);
	const configuredServerPath = inspection?.workspaceFolderLanguageValue ??
		inspection?.workspaceFolderValue ??
		inspection?.workspaceLanguageValue ??
		inspection?.workspaceValue ??
		inspection?.globalLanguageValue ??
		inspection?.globalValue;
	if (typeof configuredServerPath !== "string") {
		return undefined;
	}

	const trimmedPath = configuredServerPath.trim();
	return trimmedPath === "" ? undefined : trimmedPath;
}

async function resolveServerCommand(): Promise<ResolvedServerCommand> {
	const configuredServerPath = getExplicitServerPath();
	if (configuredServerPath) {
		const trustedCommand = await getTrustedServerPath(configuredServerPath);
		return {
			command: trustedCommand,
			expectedLocation: configuredServerPath,
		};
	}

	const bundledServerPath = getBundledServerPath();
	if (bundledServerPath) {
		return {
			command: bundledServerPath,
			expectedLocation: bundledServerPath,
		};
	}

	return {
		command: pathServerCommand,
		expectedLocation: pathServerCommand,
	};
}


async function getTrustedServerPath(configuredPath?: string): Promise<string> {
	const serverPath = validateServerPath(configuredPath ?? pathServerCommand);
	if (serverPath === pathServerCommand || isTrustedServerPath(serverPath)) {
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

function getBundledServerPath(): string | undefined {
	if (!extensionContext) {
		return undefined;
	}

	const target = bundledServerTargets.get(`${process.platform}:${process.arch}`);
	if (!target) {
		return undefined;
	}

	const binaryName = process.platform === "win32" ? "downstage.exe" : "downstage";
	const bundledPath = extensionContext.asAbsolutePath(path.join("bin", target, binaryName));
	if (!fs.existsSync(bundledPath)) {
		return undefined;
	}

	return bundledPath;
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

function getRenderPageSizeSetting(): string {
	return vscode.workspace.getConfiguration(configSection).get<string>(
		settingRenderPageSize,
		"letter",
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

async function openTemplateDocument(content: string): Promise<vscode.TextEditor> {
	const document = await vscode.workspace.openTextDocument({
		language: "downstage",
		content,
	});
	const editor = await vscode.window.showTextDocument(document, vscode.ViewColumn.One);
	const selection = findTitleValueSelection(content);
	const position = new vscode.Position(selection.line, selection.character);
	editor.selection = new vscode.Selection(position, position);
	return editor;
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


async function renderCurrentScript(styleOverride?: string): Promise<void> {
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
		const serverPath = await resolveServerCommand();
		const style = getValidatedRenderStyle(styleOverride ?? getRenderStyleSetting());
		const pageSize = getValidatedPageSize(getRenderPageSizeSetting());
		const styleName = getRenderStyleDisplayName(style);
		const pageSizeName = getPageSizeDisplayName(pageSize);
		outputChannel.appendLine(
			`Running: ${serverPath.expectedLocation} render --style ${style} --page-size ${pageSize} ${inputPath}`,
		);
		outputChannel.show(true);
		renderDiagnostics?.delete(editor.document.uri);

		await runDownstageRender(serverPath.command, style, pageSize, inputPath, outputChannel);
		const message = `Rendered ${styleName} PDF (${pageSizeName}): ${path.basename(outputPath)}`;

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
				parseRenderDiagnostics(editor.document, error.stderr, vscodeFactories) as unknown as vscode.Diagnostic[],
			);
		}
		outputChannel.appendLine(String(error));
		outputChannel.show(true);
		void vscode.window.showErrorMessage(
			"Downstage render failed. See 'Downstage Render' output for details.",
		);
	}
}

async function previewCurrentScriptPdf(styleOverride?: string): Promise<void> {
	const editor = vscode.window.activeTextEditor;
	if (!editor || editor.document.languageId !== "downstage") {
		void vscode.window.showErrorMessage("Open a Downstage script before previewing.");
		return;
	}

	if (editor.document.isUntitled) {
		void vscode.window.showErrorMessage("Save the script before previewing.");
		return;
	}

	const inputPath = editor.document.uri.fsPath;
	const outputChannel = getRenderOutputChannel();
	outputChannel.clear();

	try {
		const serverPath = await resolveServerCommand();
		const style = getValidatedRenderStyle(styleOverride ?? getRenderStyleSetting());
		const pageSize = getValidatedPageSize(getRenderPageSizeSetting());
		const styleName = getRenderStyleDisplayName(style);
		const pageSizeName = getPageSizeDisplayName(pageSize);
		const sourceName = path.basename(inputPath);
		const tempPath = path.join(
			os.tmpdir(),
			`downstage-preview-${path.basename(inputPath, ".ds")}.pdf`,
		);

		outputChannel.appendLine(
			`Running: ${serverPath.expectedLocation} render --stdin --stdout --format pdf --style ${style} --page-size ${pageSize} --source-name ${sourceName}`,
		);
		outputChannel.show(true);
		renderDiagnostics?.delete(editor.document.uri);

		await new Promise<void>((resolve, reject) => {
			let stderr = "";
			const chunks: Buffer[] = [];

			const child = spawn(serverPath.command, buildPDFPreviewArgs(style, pageSize, sourceName), {
				cwd: path.dirname(inputPath),
			});

			const timeout = setTimeout(() => {
				child.kill();
				reject(new Error("downstage render timed out after 60 seconds"));
			}, 60_000);

			child.stdout.on("data", (chunk: Buffer) => {
				chunks.push(chunk);
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
					const pdfData = Buffer.concat(chunks);
					fs.writeFileSync(tempPath, pdfData);
					previewTempFiles.add(tempPath);
					resolve();
					return;
				}
				reject(new DownstageRenderError(
					`downstage render exited with code ${code ?? "unknown"}`,
					stderr,
				));
			});

			child.stdin.on("error", () => {
				// Process may have died before we finished writing - ignore
			});
			child.stdin.write(editor.document.getText());
			child.stdin.end();
		});

		await openRenderedPdf(vscode.Uri.file(tempPath));
		void vscode.window.showInformationMessage(
			`${styleName} preview (${pageSizeName}): ${path.basename(inputPath)}`,
		);
	} catch (error) {
		if (error instanceof DownstageRenderError) {
			renderDiagnostics?.set(
				editor.document.uri,
				parseRenderDiagnostics(editor.document, error.stderr, vscodeFactories) as unknown as vscode.Diagnostic[],
			);
		}
		outputChannel.appendLine(String(error));
		outputChannel.show(true);
		void vscode.window.showErrorMessage(
			"Downstage preview failed. See 'Downstage Render' output for details.",
		);
	}
}

function getRenderOutputChannel(): vscode.OutputChannel {
	if (!renderOutputChannel) {
		renderOutputChannel = vscode.window.createOutputChannel("Downstage Render");
	}
	return renderOutputChannel;
}


async function runDownstageRender(
	serverPath: string,
	style: string,
	pageSize: string,
	inputPath: string,
	outputChannel: vscode.OutputChannel,
): Promise<void> {
	await new Promise<void>((resolve, reject) => {
		let stderr = "";
		const child = spawn(serverPath, buildPDFRenderArgs(style, pageSize, inputPath), {
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



function getPreviewDebounceSetting(): number {
	return vscode.workspace.getConfiguration(configSection).get<number>(
		settingPreviewDebounce,
		300,
	);
}

async function openLivePreview(): Promise<void> {
	let editor = vscode.window.activeTextEditor;
	if (!editor || editor.document.languageId !== "downstage") {
		editor = await openTemplateDocument(getNewPlayTemplate());
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
	let serverPathLabel: string;
	let style: string;
	try {
		const resolvedServer = await resolveServerCommand();
		serverPath = resolvedServer.command;
		serverPathLabel = resolvedServer.expectedLocation;
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
	getRenderOutputChannel().appendLine(
		`Running preview render: ${serverPathLabel} render --stdin --stdout --format html --source-anchors --style ${style} --source-name ${sourceName}`,
	);
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
					parseRenderDiagnostics(document, stderr, vscodeFactories) as unknown as vscode.Diagnostic[],
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
