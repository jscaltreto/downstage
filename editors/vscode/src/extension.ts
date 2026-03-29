import * as vscode from "vscode";
import { spawn } from "node:child_process";
import * as path from "node:path";
import {
	LanguageClient,
	LanguageClientOptions,
	ServerOptions,
	Trace,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;
const cueSuggestTimers = new Map<string, NodeJS.Timeout>();
let renderOutputChannel: vscode.OutputChannel | undefined;
let renderDiagnostics: vscode.DiagnosticCollection | undefined;

export async function activate(context: vscode.ExtensionContext): Promise<void> {
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

	context.subscriptions.push(restartCommand);
	context.subscriptions.push(renderCommand);
	context.subscriptions.push(renderCompactCommand);
	context.subscriptions.push(previewCommand);
	context.subscriptions.push(previewCompactCommand);
	context.subscriptions.push(renderDiagnostics);
	context.subscriptions.push(
		vscode.workspace.onDidChangeConfiguration(async (event) => {
			if (!event.affectsConfiguration("downstage.server.path") &&
				!event.affectsConfiguration("downstage.server.trace")) {
				return;
			}

			await restartLanguageServer(context);
		}),
	);
	context.subscriptions.push(
		vscode.workspace.onDidChangeTextDocument((event) => {
			scheduleCueSuggestForDocument(event.document);
		}),
	);
	context.subscriptions.push(
		vscode.window.onDidChangeTextEditorSelection((event) => {
			scheduleCueSuggestForEditor(event.textEditor);
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
	const serverPath = getServerPath();
	const outputChannel = vscode.window.createOutputChannel("Downstage Language Server");
	context.subscriptions.push(outputChannel);

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

	try {
		await client.start();
	} catch (error) {
		client = undefined;
		const message = [
			"Failed to start the Downstage language server.",
			`Expected executable: ${serverPath}`,
			"Install the `downstage` binary or set `downstage.server.path`.",
		].join(" ");
		outputChannel.appendLine(String(error));
		void vscode.window.showErrorMessage(message, "Open Settings").then((selection) => {
			if (selection === "Open Settings") {
				void vscode.commands.executeCommand(
					"workbench.action.openSettings",
					"downstage.server.path",
				);
			}
		});
	}
}

function getServerPath(): string {
	return vscode.workspace.getConfiguration("downstage").get<string>("server.path", "downstage");
}

function getTraceSetting(): string {
	return vscode.workspace.getConfiguration("downstage").get<string>("server.trace", "off");
}

function getAutoSuggestSetting(): boolean {
	return vscode.workspace.getConfiguration("downstage").get<boolean>(
		"editor.autoSuggestCharacterCues",
		true,
	);
}

function getRenderStyleSetting(): string {
	return vscode.workspace.getConfiguration("downstage").get<string>(
		"render.style",
		"standard",
	);
}

function getOpenAfterRenderSetting(): boolean {
	return vscode.workspace.getConfiguration("downstage").get<boolean>(
		"render.openAfterRender",
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

	const serverPath = getServerPath();
	const style = styleOverride ?? getRenderStyleSetting();
	const inputPath = editor.document.uri.fsPath;
	const outputPath = replaceExtension(inputPath, ".pdf");
	const outputChannel = getRenderOutputChannel();

	outputChannel.clear();
	outputChannel.appendLine(`Running: ${serverPath} render --style ${style} ${inputPath}`);
	outputChannel.show(true);
	renderDiagnostics?.delete(editor.document.uri);

	try {
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

class DownstageRenderError extends Error {
	readonly stderr: string;

	constructor(message: string, stderr: string) {
		super(message);
		this.stderr = stderr;
	}
}
