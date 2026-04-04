import { chmodSync, cpSync, existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { dirname, isAbsolute, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const extensionDir = resolve(scriptDir, "..");
const repoRoot = resolve(extensionDir, "..", "..");
const stageDir = join(extensionDir, ".vsix-stage");
const packageDir = join(stageDir, "extension");
const supportedTargets = new Set(["linux-x64", "darwin-x64", "darwin-arm64", "win32-x64"]);
const options = parseArgs(process.argv.slice(2));

if (!options.skipCompile) {
	run("npm", ["run", "compile"], extensionDir);
}

rmSync(stageDir, { recursive: true, force: true });
mkdirSync(packageDir, { recursive: true });

const manifestPath = join(extensionDir, "package.json");
const manifest = JSON.parse(readFileSync(manifestPath, "utf8"));
const packageVersion = normalizeVersion(options.version ?? manifest.version);
const packageFiles = [
	"out",
	"syntaxes",
	"snippets",
	"images",
	"language-configuration.json",
	"README.md",
	"CHANGELOG.md",
	"LICENSE",
	"SUPPORT.md",
];
const stagedManifest = {
	name: manifest.name,
	displayName: manifest.displayName,
	description: manifest.description,
	version: packageVersion,
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
	files: packageFiles,
};

copy("README.md");
copyFromRepo("LICENSE");
copy("language-configuration.json");
copyBundledOutput();
copy("syntaxes");
copy("snippets");
copy("images");
copy("SUPPORT.md");
copyFromRepo("CHANGELOG.md");

if (options.binary) {
	if (!options.target) {
		throw new Error("--binary requires --target");
	}
	copyBundledBinary(options.target, options.binary);
	packageFiles.push("bin");
}

writeFileSync(join(packageDir, "package.json"), JSON.stringify(stagedManifest, null, 2) + "\n");

const vsceBin = join(extensionDir, "node_modules", ".bin", "vsce");
const packageName = getPackageName(packageVersion, options.target);
const packageArgs = ["package", "--out", join(extensionDir, packageName)];
if (options.target) {
	packageArgs.push("--target", options.target);
}
run(vsceBin, packageArgs, packageDir);

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

function copyBundledBinary(target, binaryPath) {
	const binaryName = target.startsWith("win32-") ? "downstage.exe" : "downstage";
	const destinationDir = join(packageDir, "bin", target);
	const destinationPath = join(destinationDir, binaryName);
	mkdirSync(destinationDir, { recursive: true });
	copyPath(binaryPath, destinationPath, `bundled binary for ${target}`);
	if (!target.startsWith("win32-")) {
		chmodSync(destinationPath, 0o755);
	}
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

function parseArgs(args) {
	/** @type {{ target?: string; binary?: string; version?: string; skipCompile: boolean }} */
	const options = {};

	for (let index = 0; index < args.length; index += 1) {
		const argument = args[index];
		if (argument === "--skip-compile") {
			options.skipCompile = true;
			continue;
		}
		if (argument === "--target") {
			options.target = getArgValue(argument, args[index + 1]);
			index += 1;
			continue;
		}
		if (argument === "--binary") {
			options.binary = resolveInputPath(getArgValue(argument, args[index + 1]));
			index += 1;
			continue;
		}
		if (argument === "--version") {
			options.version = getArgValue(argument, args[index + 1]);
			index += 1;
			continue;
		}

		throw new Error(`Unknown argument: ${argument}`);
	}

	if (options.target && !supportedTargets.has(options.target)) {
		throw new Error(`Unsupported target: ${options.target}`);
	}

	return options;
}

function getArgValue(flag, value) {
	if (!value || value.startsWith("--")) {
		throw new Error(`Missing value for ${flag}`);
	}

	return value;
}

function resolveInputPath(inputPath) {
	if (isAbsolute(inputPath)) {
		return inputPath;
	}

	const repoRelativePath = resolve(repoRoot, inputPath);
	if (existsSync(repoRelativePath)) {
		return repoRelativePath;
	}

	return resolve(extensionDir, inputPath);
}

function normalizeVersion(version) {
	const normalizedVersion = version.startsWith("v") ? version.slice(1) : version;
	if (!/^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$/.test(normalizedVersion)) {
		throw new Error(`Invalid extension version: ${version}`);
	}

	return normalizedVersion;
}

function getPackageName(version, target) {
	if (!target) {
		return `downstage-vscode-${version}.vsix`;
	}

	return `downstage-vscode-${version}-${target}.vsix`;
}
