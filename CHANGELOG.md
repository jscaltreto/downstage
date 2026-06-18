# Changelog

## [0.9.0](https://github.com/jscaltreto/downstage/compare/v0.8.0...v0.9.0) (2026-06-12)


### Features

* **desktop:** Downstage Write — Wails-based desktop app (alpha) ([#188](https://github.com/jscaltreto/downstage/issues/188)) ([fb1f943](https://github.com/jscaltreto/downstage/commit/fb1f943a2350ebc47920b709ad216937bbb74454))
* **desktop:** sidebar delete/restore + library-wide change reconciliation (Track A2) ([#216](https://github.com/jscaltreto/downstage/issues/216)) ([ac57e30](https://github.com/jscaltreto/downstage/commit/ac57e30225ac9747cec76b68f6420c36edbfc0ac))
* **revisions:** add revision-pages workflow (`downstage revisions`) ([#187](https://github.com/jscaltreto/downstage/issues/187)) ([ffa6955](https://github.com/jscaltreto/downstage/commit/ffa6955352294380fd53b172fb74ba72f0ee885c))
* **web:** structured Help drawer (Track B of [#189](https://github.com/jscaltreto/downstage/issues/189)) ([#218](https://github.com/jscaltreto/downstage/issues/218)) ([b0d9487](https://github.com/jscaltreto/downstage/commit/b0d9487aef669243a68c848ecfe4a9c7292f5272))


### Bug Fixes

* **deps:** bump liquidjs to 10.27.0 to clear three security advisories ([#186](https://github.com/jscaltreto/downstage/issues/186)) ([d3ced40](https://github.com/jscaltreto/downstage/commit/d3ced403adbff2d112c7ba7f128b6e7d523187c0))
* **pdf:** avoid O(N²) form-XObject duplication in imposed PDFs ([#181](https://github.com/jscaltreto/downstage/issues/181)) ([7f7973a](https://github.com/jscaltreto/downstage/commit/7f7973ac461a7c34fc9bb7dd0b67e74789c67779))


### Dependencies

* **deps:** bump @playwright/test from 1.59.1 to 1.60.0 in /web ([#200](https://github.com/jscaltreto/downstage/issues/200)) ([93536da](https://github.com/jscaltreto/downstage/commit/93536da297bf6d5153914f2c0ee857e3d019d3b6))
* **deps:** bump @tailwindcss/postcss from 4.2.2 to 4.3.0 in /web ([#198](https://github.com/jscaltreto/downstage/issues/198)) ([ce08ae6](https://github.com/jscaltreto/downstage/commit/ce08ae63b3daf8e7c26cc989fa56dd07a9c40fab))
* **deps:** bump @types/node from 25.6.0 to 25.9.0 in /editors/vscode ([#195](https://github.com/jscaltreto/downstage/issues/195)) ([f450721](https://github.com/jscaltreto/downstage/commit/f450721f958d514dccaa1c8e75b13531bc160314))
* **deps:** bump @types/vscode in /editors/vscode ([#178](https://github.com/jscaltreto/downstage/issues/178)) ([3363d75](https://github.com/jscaltreto/downstage/commit/3363d75d49704078a2df74e92202d66810b58a88))
* **deps:** bump @vscode/vsce from 3.7.1 to 3.9.1 in /editors/vscode ([#163](https://github.com/jscaltreto/downstage/issues/163)) ([19e157b](https://github.com/jscaltreto/downstage/commit/19e157bf775010483939942c15b2ba481b1f6149))
* **deps:** bump actions/cache from 4 to 5 ([#215](https://github.com/jscaltreto/downstage/issues/215)) ([65ae1fc](https://github.com/jscaltreto/downstage/commit/65ae1fcf4a1258fbd5cf42e4b9c5b12a1455f8cd))
* **deps:** bump actions/dependency-review-action from 4.9.0 to 5.0.0 ([#202](https://github.com/jscaltreto/downstage/issues/202)) ([3c2f641](https://github.com/jscaltreto/downstage/commit/3c2f64102a2d111fc334a6620395e3071a8ca0b7))
* **deps:** bump actions/setup-go from 6.3.0 to 6.4.0 ([#162](https://github.com/jscaltreto/downstage/issues/162)) ([a0ff936](https://github.com/jscaltreto/downstage/commit/a0ff936d467e18c5d391417607066f5e8f8b7952))
* **deps:** bump actions/upload-artifact from 4 to 7 ([#174](https://github.com/jscaltreto/downstage/issues/174)) ([3b893e9](https://github.com/jscaltreto/downstage/commit/3b893e9e06a5318421db39c3f88fd356ee4ca015))
* **deps:** bump fast-uri from 3.1.0 to 3.1.2 in /editors/vscode ([#180](https://github.com/jscaltreto/downstage/issues/180)) ([afd730b](https://github.com/jscaltreto/downstage/commit/afd730b8d738ee7c2704d6b5372fedcb56db24c5))
* **deps:** bump github/codeql-action from 3.28.18 to 4.35.5 ([#201](https://github.com/jscaltreto/downstage/issues/201)) ([9502757](https://github.com/jscaltreto/downstage/commit/9502757983b1cb4a0bc60f4eba2b3d5da387a5a4))
* **deps:** bump github/codeql-action from 4.35.5 to 4.36.0 ([#214](https://github.com/jscaltreto/downstage/issues/214)) ([f94d251](https://github.com/jscaltreto/downstage/commit/f94d251012f81e826576aa96b88341004dc8e608))
* **deps:** bump googleapis/release-please-action from 4.4.0 to 5.0.0 ([#175](https://github.com/jscaltreto/downstage/issues/175)) ([ae847ea](https://github.com/jscaltreto/downstage/commit/ae847ea16ece20d9fad99c1d8c4adbd79991d177))
* **deps:** bump goreleaser/goreleaser-action from 7.0.0 to 7.2.2 ([#203](https://github.com/jscaltreto/downstage/issues/203)) ([33c838e](https://github.com/jscaltreto/downstage/commit/33c838ea27db9ec44d882e0f32a003801487d771))
* **deps:** bump postcss from 8.5.8 to 8.5.14 ([#184](https://github.com/jscaltreto/downstage/issues/184)) ([83ac2f8](https://github.com/jscaltreto/downstage/commit/83ac2f85d22dc720ffefc32f2c5a7bb36d4af4af))
* **deps:** bump qs from 6.15.0 to 6.15.2 in /editors/vscode ([#193](https://github.com/jscaltreto/downstage/issues/193)) ([e59d904](https://github.com/jscaltreto/downstage/commit/e59d904b757464b844f596fffd935d61bb7773da))
* **deps:** bump sigstore/cosign-installer in the actions-patch group ([#213](https://github.com/jscaltreto/downstage/issues/213)) ([f91efe4](https://github.com/jscaltreto/downstage/commit/f91efe4db784a3a604e9cb0e9cffc1221a751c6d))
* **deps:** bump tailwindcss from 4.2.2 to 4.3.0 in /web ([#197](https://github.com/jscaltreto/downstage/issues/197)) ([1a58ed6](https://github.com/jscaltreto/downstage/commit/1a58ed6fefea1bcd6cd222dc5b67978ad6eab45a))
* **deps:** bump the vscode-patch group ([#209](https://github.com/jscaltreto/downstage/issues/209)) ([ed5018c](https://github.com/jscaltreto/downstage/commit/ed5018c441eae0ad559dab348db96b3a6c87bc2c))
* **deps:** bump the web-patch group across 1 directory with 10 updates ([#196](https://github.com/jscaltreto/downstage/issues/196)) ([513e70e](https://github.com/jscaltreto/downstage/commit/513e70e03bf75a5f07d47f4c6f722a846b4b6fde))
* **deps:** bump the web-patch group in /web with 2 updates ([#210](https://github.com/jscaltreto/downstage/issues/210)) ([d4840c3](https://github.com/jscaltreto/downstage/commit/d4840c3aa49f58ddb736dfcae9360a7ba3da1e43))
* **deps:** bump tmp from 0.2.5 to 0.2.7 in /editors/vscode ([#208](https://github.com/jscaltreto/downstage/issues/208)) ([c516c3c](https://github.com/jscaltreto/downstage/commit/c516c3ce5583da7eb93e2a4139d3a0ce94244ee1))
* **deps:** bump uuid and @azure/msal-node in /editors/vscode ([#179](https://github.com/jscaltreto/downstage/issues/179)) ([0da7665](https://github.com/jscaltreto/downstage/commit/0da76650d30587d1e81f26372a375a11afb70ad5))
* **deps:** bump vitest from 4.1.4 to 4.1.6 in /editors/vscode ([#177](https://github.com/jscaltreto/downstage/issues/177)) ([7235da6](https://github.com/jscaltreto/downstage/commit/7235da66671e1f48ccdf76e461f3781dedfa92fc))
* **deps:** bump vue in /web in the web-patch group ([#217](https://github.com/jscaltreto/downstage/issues/217)) ([93d2ce0](https://github.com/jscaltreto/downstage/commit/93d2ce03bdb7ea76d21b6e9888d32b89a322135d))

## [0.8.0](https://github.com/jscaltreto/downstage/compare/v0.7.0...v0.8.0) (2026-04-20)


### ⚠ BREAKING CHANGES

* **pdf:** the web/WASM renderPDF signature preferred shape is now renderPDF(source, { style, pageSize, layout, gutter }). The positional (source, style?, pageSize?) form is still accepted internally for one release but is deprecated; external callers should migrate.
* **pdf:** the render.PageA4 constant value is now "a4" (was "A4"). External Go callers that assigned the raw string "A4" directly to Config.PageSize must switch to render.PageA4 or ParsePageSize.

### Features

* **pdf:** add 2-up and booklet layouts for acting edition ([#171](https://github.com/jscaltreto/downstage/issues/171)) ([e0afbe4](https://github.com/jscaltreto/downstage/commit/e0afbe43e57b2de9518a8b0ed0c30dd6f2ed587a))
* **pdf:** add Letter and A4 page sizes across export surfaces ([#170](https://github.com/jscaltreto/downstage/issues/170)) ([793f0bb](https://github.com/jscaltreto/downstage/commit/793f0bbc8927ea86cebc09010243bb8528ac61e7))


### Bug Fixes

* **render:** drop blank page between title page and generic section in condensed ([#172](https://github.com/jscaltreto/downstage/issues/172)) ([6bca011](https://github.com/jscaltreto/downstage/commit/6bca011c664963db1f4cbc97b951640ede4bff61))
* **web:** pin empty PostCSS config so vite stops walking up the tree ([#168](https://github.com/jscaltreto/downstage/issues/168)) ([1c7bce3](https://github.com/jscaltreto/downstage/commit/1c7bce353ff1e5cf98109f9b9afac9177e49cfee))

## [0.7.0](https://github.com/jscaltreto/downstage/compare/v0.6.0...v0.7.0) (2026-04-16)


### Features

* **lexer:** require blank line before cue ([#152](https://github.com/jscaltreto/downstage/issues/152)) ([ba33d2a](https://github.com/jscaltreto/downstage/commit/ba33d2a59b3bdeda16262310c424e5678e7934d0))
* **lsp:** add DP and cue hygiene diagnostics ([#147](https://github.com/jscaltreto/downstage/issues/147)) ([56c8b27](https://github.com/jscaltreto/downstage/commit/56c8b27e0ba9f8b954eaf77b15793536d40d1ca7))
* **lsp:** add structural Rename Symbol for characters ([#155](https://github.com/jscaltreto/downstage/issues/155)) ([ad485e0](https://github.com/jscaltreto/downstage/commit/ad485e044d65e8f9f147aa8b49976e921296c3a7))
* **lsp:** suppress unknown-character diagnostic for forced cues ([#144](https://github.com/jscaltreto/downstage/issues/144)) ([fe92efa](https://github.com/jscaltreto/downstage/commit/fe92efaa503128ad58c25ca4df98eceec1eddfde))
* **stats:** add manuscript statistics command ([#158](https://github.com/jscaltreto/downstage/issues/158)) ([3a0c965](https://github.com/jscaltreto/downstage/commit/3a0c965da0ec796899769b309206bd68a6552d3b))
* **web:** add find/replace to editor ([#149](https://github.com/jscaltreto/downstage/issues/149)) ([ddff1ee](https://github.com/jscaltreto/downstage/commit/ddff1eea124d7f25611ad93969ff2dfd11528d26))
* **web:** add issues drawer with status FAB ([#146](https://github.com/jscaltreto/downstage/issues/146)) ([7f54afd](https://github.com/jscaltreto/downstage/commit/7f54afdee26de94e18e61892cdce01e84421cf30))
* **web:** add outline tab + scroll-sync fixes + FAB polish ([#151](https://github.com/jscaltreto/downstage/issues/151)) ([b09c602](https://github.com/jscaltreto/downstage/commit/b09c6021865f5450e27e6119d60c34276301d453))
* **web:** add spell check ([#142](https://github.com/jscaltreto/downstage/issues/142)) ([0212401](https://github.com/jscaltreto/downstage/commit/021240198fa7b6192ab4097d5995d4354a906542))
* **web:** add stats drawer tab and toolbar toggle ([#159](https://github.com/jscaltreto/downstage/issues/159)) ([f3410cd](https://github.com/jscaltreto/downstage/commit/f3410cd669b7aa2f203f9ac1b52c940fb811ff59))
* **web:** replace quick reference with help drawer ([#160](https://github.com/jscaltreto/downstage/issues/160)) ([c66fc7e](https://github.com/jscaltreto/downstage/commit/c66fc7e5d4fa489115b285e37ae80eea4c5f3116))


### Bug Fixes

* **lsp:** target missing dp edit by play ([#154](https://github.com/jscaltreto/downstage/issues/154)) ([ceb722d](https://github.com/jscaltreto/downstage/commit/ceb722d10bcb3c331babe5ca6885a642bb44e853))
* **render:** stylable character descriptions and frontmatter values ([#153](https://github.com/jscaltreto/downstage/issues/153)) ([de0399b](https://github.com/jscaltreto/downstage/commit/de0399b04165ea28a6069bc9bcb82adf361328a7))
* **render:** use dp heading wording ([#148](https://github.com/jscaltreto/downstage/issues/148)) ([5cef99c](https://github.com/jscaltreto/downstage/commit/5cef99cbddb1ea886a00d5641dd620ee5690f0ba))
* **web:** avoid caret misplacement on wrapped lintRange lines ([#150](https://github.com/jscaltreto/downstage/issues/150)) ([fbf3909](https://github.com/jscaltreto/downstage/commit/fbf3909a90b67f0978476c65275681909bc086a9))

## [0.6.0](https://github.com/jscaltreto/downstage/compare/v0.5.0...v0.6.0) (2026-04-13)


### ⚠ BREAKING CHANGES

* V1 documents no longer render without migration.

### Features

* scoped V2 document model and migration ([#131](https://github.com/jscaltreto/downstage/issues/131)) ([04fedd9](https://github.com/jscaltreto/downstage/commit/04fedd9ee46f66e359a810f4c40b0b39e920887e))
* **web:** wire LSP completions and code actions into editor ([#129](https://github.com/jscaltreto/downstage/issues/129)) ([0da00bd](https://github.com/jscaltreto/downstage/commit/0da00bd03755a78a6c9236358fe58a939bdc873f))


### Bug Fixes

* **ci:** drop release-please package-name so merged release PRs tag ([#136](https://github.com/jscaltreto/downstage/issues/136)) ([d700647](https://github.com/jscaltreto/downstage/commit/d7006476f3efa4e58dd39571bbab20d0847ef44e))
* **ci:** drop separate-pull-requests so release PR merge can tag ([#140](https://github.com/jscaltreto/downstage/issues/140)) ([f9aaa8e](https://github.com/jscaltreto/downstage/commit/f9aaa8e00322766fb6b9c2f643ca7d011a6b30c9))
* **ci:** restore release-please baseline and surface component in PR title ([#138](https://github.com/jscaltreto/downstage/issues/138)) ([514199b](https://github.com/jscaltreto/downstage/commit/514199b813aaec3624307aee27bd4a5543e82fe1))

## [0.6.0](https://github.com/jscaltreto/downstage/compare/v0.5.0...v0.6.0) (2026-04-13)


### ⚠ BREAKING CHANGES

* V1 documents no longer render without migration.

### Features

* scoped V2 document model and migration ([#131](https://github.com/jscaltreto/downstage/issues/131)) ([04fedd9](https://github.com/jscaltreto/downstage/commit/04fedd9ee46f66e359a810f4c40b0b39e920887e))
* **web:** wire LSP completions and code actions into editor ([#129](https://github.com/jscaltreto/downstage/issues/129)) ([0da00bd](https://github.com/jscaltreto/downstage/commit/0da00bd03755a78a6c9236358fe58a939bdc873f))

## [0.5.0](https://github.com/jscaltreto/downstage/compare/v0.4.1...v0.5.0) (2026-04-12)


### Features

* **web:** add first-run welcome modal ([#126](https://github.com/jscaltreto/downstage/issues/126)) ([1cd4535](https://github.com/jscaltreto/downstage/commit/1cd453598956d2dfcaba0f17db9f198282ea5af8))
* **web:** refactor editor with Vue and Tailwind ([#123](https://github.com/jscaltreto/downstage/issues/123)) ([e69b68c](https://github.com/jscaltreto/downstage/commit/e69b68c3f376bed215105173e903d61384a998a5))


### Bug Fixes

* **pdf:** paginate dialogue continuations ([#110](https://github.com/jscaltreto/downstage/issues/110)) ([6b86063](https://github.com/jscaltreto/downstage/commit/6b86063ef9b6f2fa1e2827c0dc19001e51e3609a))
* tighten condensed layout and formatting ([#120](https://github.com/jscaltreto/downstage/issues/120)) ([96e3c02](https://github.com/jscaltreto/downstage/commit/96e3c025edbf030e9e43eeffa42387de0cc8ba81))

## [0.4.1](https://github.com/jscaltreto/downstage/compare/v0.4.0...v0.4.1) (2026-04-05)


### Bug Fixes

* **site:** make web editor primary onboarding path ([#99](https://github.com/jscaltreto/downstage/issues/99)) ([fd14adc](https://github.com/jscaltreto/downstage/commit/fd14adcbf6be3faf4545892c6a7c9d82657c3cf4)), closes [#92](https://github.com/jscaltreto/downstage/issues/92)

## [0.4.0](https://github.com/jscaltreto/downstage/compare/v0.3.0...v0.4.0) (2026-04-05)


### Features

* **vscode:** guide first-run playwriting flow ([#95](https://github.com/jscaltreto/downstage/issues/95)) ([7d49417](https://github.com/jscaltreto/downstage/commit/7d494174643aca2594b22cf83b8915208856a42f))
* **web:** improve browser editor onboarding ([#97](https://github.com/jscaltreto/downstage/issues/97)) ([68a4c61](https://github.com/jscaltreto/downstage/commit/68a4c617561f66eba08a060e95effd1a62ab0164))

## [0.3.0](https://github.com/jscaltreto/downstage/compare/v0.2.0...v0.3.0) (2026-04-05)


### Features

* add VS Code extension MVP ([#13](https://github.com/jscaltreto/downstage/issues/13)) ([04d3e6e](https://github.com/jscaltreto/downstage/commit/04d3e6e01baafdee58a94136320e67ed6196b042))
* **callouts:** support &gt;&gt; scene callouts ([#61](https://github.com/jscaltreto/downstage/issues/61)) ([d92c6b4](https://github.com/jscaltreto/downstage/commit/d92c6b414a6ffa060f7c459b8f61770669d4cf15))
* live web editor with WASM-compiled Downstage ([#80](https://github.com/jscaltreto/downstage/issues/80)) ([280e72e](https://github.com/jscaltreto/downstage/commit/280e72e1a2a6011163ccc7b7b265178b61d2a832))
* **lsp:** add scene-aware cue completion ([#11](https://github.com/jscaltreto/downstage/issues/11)) ([dcc245b](https://github.com/jscaltreto/downstage/commit/dcc245ba32fe17fb6631bab136466fb653df7513))
* **lsp:** diagnose misnumbered headings ([#37](https://github.com/jscaltreto/downstage/issues/37)) ([b093d5f](https://github.com/jscaltreto/downstage/commit/b093d5f938342873927edab502f796e4dc002bdf))
* **lsp:** relax unknown-character warnings for collective and conjunction cues ([#36](https://github.com/jscaltreto/downstage/issues/36)) ([44ca0b1](https://github.com/jscaltreto/downstage/commit/44ca0b1193126a96b5897e5b841bbb7733490616)), closes [#28](https://github.com/jscaltreto/downstage/issues/28)
* **lsp:** warn on unnumbered acts and scenes ([#27](https://github.com/jscaltreto/downstage/issues/27)) ([d4fc2c4](https://github.com/jscaltreto/downstage/commit/d4fc2c45cc53edb88a527b85b2f6ed2f3090d7d2))
* **render:** add HTML rendering support ([#15](https://github.com/jscaltreto/downstage/issues/15)) ([949d324](https://github.com/jscaltreto/downstage/commit/949d32423f97c1e6960d08e98c7ea21c8d39b8b0))
* **vscode:** add live HTML preview with scroll sync ([#29](https://github.com/jscaltreto/downstage/issues/29)) ([afb31e0](https://github.com/jscaltreto/downstage/commit/afb31e0ba66cd9fb9d8977024b7dbf5dd3b302af))
* **vscode:** bundle release binaries ([#63](https://github.com/jscaltreto/downstage/issues/63)) ([6abd2e0](https://github.com/jscaltreto/downstage/commit/6abd2e0b2e25653dd4280b683634f6184d2f0f5b))


### Bug Fixes

* address audit quick wins across LSP, CLI, and CI ([#25](https://github.com/jscaltreto/downstage/issues/25)) ([1053815](https://github.com/jscaltreto/downstage/commit/10538153cb99fad93e343080ff2398bbc3cce230))
* harden input validation and resource limits ([#35](https://github.com/jscaltreto/downstage/issues/35)) ([c4895d2](https://github.com/jscaltreto/downstage/commit/c4895d26208f52859e3448c8636fe694833ac930))
* polish homepage copy and content styling ([#85](https://github.com/jscaltreto/downstage/issues/85)) ([4ffa19f](https://github.com/jscaltreto/downstage/commit/4ffa19f57463b9530b8ca7adb4a599391cce1701))
* **render:** paragraph spacing in dialogue and condensed stage directions ([#51](https://github.com/jscaltreto/downstage/issues/51)) ([a36dc52](https://github.com/jscaltreto/downstage/commit/a36dc528ae0291ae4e62c2870ce74e349e6f434f))
* **render:** refine stage direction spacing based on adjacency ([#52](https://github.com/jscaltreto/downstage/issues/52)) ([e7b7db2](https://github.com/jscaltreto/downstage/commit/e7b7db21e8f8936df52aa5bda90055d8ed325366))
* **render:** render parentheticals on separate line in HTML output ([#32](https://github.com/jscaltreto/downstage/issues/32)) ([dc96248](https://github.com/jscaltreto/downstage/commit/dc96248860e93ce42c8651a99d1cfa785b641be5))
* **render:** support --stdout for PDF output ([#60](https://github.com/jscaltreto/downstage/issues/60)) ([5536997](https://github.com/jscaltreto/downstage/commit/5536997dc48c1477a3c6e065e66919830ce5c4d9))
* **render:** tighten compact stage gap ([#57](https://github.com/jscaltreto/downstage/issues/57)) ([9ec43a4](https://github.com/jscaltreto/downstage/commit/9ec43a455ea625b3ee8490293fd4719d360ad65d))
* repair site links and docs CTA styling ([#84](https://github.com/jscaltreto/downstage/issues/84)) ([c28d1e0](https://github.com/jscaltreto/downstage/commit/c28d1e0fab535d6c7a7ce77243e13435f3963f0a))
* **site:** cache bust editor assets ([#83](https://github.com/jscaltreto/downstage/issues/83)) ([eb5993f](https://github.com/jscaltreto/downstage/commit/eb5993fd359bec106c6e6533b45bf0bc3582ed40))
* **site:** respect Pages path prefix ([#79](https://github.com/jscaltreto/downstage/issues/79)) ([0ce088a](https://github.com/jscaltreto/downstage/commit/0ce088a03566f62244070755836e5ba54683d736))
* **vscode:** double-buffer preview to eliminate update flash ([#43](https://github.com/jscaltreto/downstage/issues/43)) ([4c034b6](https://github.com/jscaltreto/downstage/commit/4c034b6e31e04a41ffdeb9884e4d2a95ff906fe8))
* **vscode:** stop live preview scroll flash ([#38](https://github.com/jscaltreto/downstage/issues/38)) ([9384309](https://github.com/jscaltreto/downstage/commit/93843095b9ddfcb300b0baf3a8dad97b08051f4c))
* wait for web editor wasm api startup ([#82](https://github.com/jscaltreto/downstage/issues/82)) ([675182d](https://github.com/jscaltreto/downstage/commit/675182de89a48c238819b2a23713f1c9d011d91b))
* **web:** harden wasm startup ([#81](https://github.com/jscaltreto/downstage/issues/81)) ([372e8cb](https://github.com/jscaltreto/downstage/commit/372e8cb4e67d776b54372d3214a50dd0cf9331e3))


### Performance Improvements

* **lsp:** cache parsed document indexes ([#31](https://github.com/jscaltreto/downstage/issues/31)) ([2f5f7d1](https://github.com/jscaltreto/downstage/commit/2f5f7d1a727d7331aaa49e227e6d718f4871703f))

## [0.2.0](https://github.com/jscaltreto/downstage/compare/v0.1.0...v0.2.0) (2026-03-29)


### Features

* **language:** add dual dialogue support ([#9](https://github.com/jscaltreto/downstage/issues/9)) ([cbda3d1](https://github.com/jscaltreto/downstage/commit/cbda3d148ec7e2359a5cb9ebc1e2ec62e3755656))


### Bug Fixes

* **ci:** correct pages action pins ([#6](https://github.com/jscaltreto/downstage/issues/6)) ([762a90f](https://github.com/jscaltreto/downstage/commit/762a90fe0e9f5059be79810f64f1a08df8b05814))

## 0.1.0 (2026-03-29)


### Features

* initialize downstage ([b60ad38](https://github.com/jscaltreto/downstage/commit/b60ad3836ee77d23945aaf02231f8ae2fc4c9156))


### Miscellaneous Chores

* release 0.1.0 ([0b9b680](https://github.com/jscaltreto/downstage/commit/0b9b68026bd0eec779b641a115502050461860ef))

## Changelog

All notable changes to this project will be documented in this file.

This file is managed by Release Please.
