# Changelog

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
