import { cpSync, existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const extensionDir = resolve(scriptDir, "..");
const repoRoot = resolve(extensionDir, "..", "..");
const stageDir = join(extensionDir, ".vsix-stage");
const packageDir = join(stageDir, "extension");

run("npm", ["run", "compile"], extensionDir);

rmSync(stageDir, { recursive: true, force: true });
mkdirSync(packageDir, { recursive: true });

const manifestPath = join(extensionDir, "package.json");
const manifest = JSON.parse(readFileSync(manifestPath, "utf8"));
const stagedManifest = {
	name: manifest.name,
	displayName: manifest.displayName,
	description: manifest.description,
	version: manifest.version,
	publisher: manifest.publisher,
	license: manifest.license,
	icon: manifest.icon,
	galleryBanner: manifest.galleryBanner,
	homepage: manifest.homepage,
	repository: manifest.repository,
	bugs: manifest.bugs,
	markdown: manifest.markdown,
	pricing: manifest.pricing,
	qna: manifest.qna,
	engines: manifest.engines,
	categories: manifest.categories,
	keywords: manifest.keywords,
	main: manifest.main,
	contributes: manifest.contributes,
	files: ["out", "syntaxes", "snippets", "images", "language-configuration.json", "README.md", "CHANGELOG.md", "LICENSE", "SUPPORT.md"],
};

writeFileSync(join(packageDir, "package.json"), JSON.stringify(stagedManifest, null, 2) + "\n");

copy("README.md");
copyFromRepo("LICENSE");
copy("language-configuration.json");
copyBundledOutput();
copy("syntaxes");
copy("snippets");
copy("images");
copy("SUPPORT.md");
copyFromRepo("CHANGELOG.md");

const vsceBin = join(extensionDir, "node_modules", ".bin", "vsce");
const packageName = `downstage-vscode-${manifest.version}.vsix`;
run(vsceBin, ["package", "--out", join(extensionDir, packageName)], packageDir);

function copy(relativePath) {
	const source = join(extensionDir, relativePath);
	copyPath(source, join(packageDir, relativePath), relativePath);
}

function copyFromRepo(relativePath) {
	const source = join(repoRoot, relativePath);
	copyPath(source, join(packageDir, relativePath), relativePath);
}

function copyBundledOutput() {
	mkdirSync(join(packageDir, "out"), { recursive: true });
	copyPath(
		join(extensionDir, "out", "extension.js"),
		join(packageDir, "out", "extension.js"),
		"out/extension.js",
	);
}

function copyPath(source, destination, label) {
	if (!existsSync(source)) {
		throw new Error(`Missing required file or directory: ${label}`);
	}

	cpSync(source, destination, { recursive: true });
}

function run(command, args, cwd) {
	const result = spawnSync(command, args, {
		cwd,
		stdio: "inherit",
	});

	if (result.error) {
		console.error(`Failed to run ${command}: ${result.error.message}`);
		process.exit(1);
	}

	if (result.status !== 0) {
		process.exit(result.status ?? 1);
	}
}
