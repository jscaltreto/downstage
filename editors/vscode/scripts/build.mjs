import esbuild from "esbuild";

const watch = process.argv.includes("--watch");

const context = await esbuild.context({
	entryPoints: ["src/extension.ts"],
	outfile: "out/extension.js",
	bundle: true,
	platform: "node",
	format: "cjs",
	target: "node20",
	sourcemap: true,
	external: ["vscode"],
	logLevel: "info",
});

if (watch) {
	await context.watch();
	console.log("Watching Downstage VS Code extension...");
} else {
	await context.rebuild();
	await context.dispose();
}
