# Changelog

Notable changes to `go-smapiweb`. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-11-21

### Fixed

- A cross-origin HTTP redirect is refused.
- Replacement values are dropped when they are unparseable, network-path, protocol-relative, non-web or carry control characters; plain names `url.Parse` rejects are kept.
- A compat document whose mods are the wrong shape is rejected, one wrong-typed field fails the whole compat list or update answer, and an unrecognised status maps to unknown, not ok.
- Unofficial status is inferred from the version alone, `ByNexus` is dropped when a page names two mods, and `PageURL` goes through `webURL`.
- An update answer body that is not a list is rejected, an answer without an id is skipped instead of failing the batch, and answers from a successful retry are kept.
- `Index` keeps `Nexus:0` and negative page ids.

### Added

- `CompatEntry.RawStatus` keeps the status text as the API sent it, `StatusUnknown` marks one that is not recognised, and `CrossOriginRedirectError` is what a refused cross-origin redirect returns.

## [0.1.1] - 2026-11-07

### Fixed

- `UploadLog` refuses a `Location` on another scheme or host than the request.
- `Update.URL`, `UnofficialURL` and an unofficial update's URL are kept only when they are http or https addresses.
- A 307 or 308 redirect to another origin is refused, so the update ask's body is not re-sent there; an https-to-http redirect is refused.
- `Page` blanks file names `.`, `..` and those holding a colon or control character, as it already did for separators.
- `Index` keeps at most 64 distinct pages per UniqueID.

## [0.1.0] - 2026-10-29

### Added

- `Client.Index` and `Client.Page` read the StardewModDataset: which Nexus pages ship a UniqueID, and a page's files, manifests and dependencies. A page the dataset lacks is an empty `Page`.
- `Client.CompatList` reads the SMAPI compatibility list in every shape it has been published in, into a `CompatIndex` with `Lookup` by UniqueID or Nexus page and `CompatEntry.BrokenOn` for a game version.

[Unreleased]: https://github.com/Rethunk-Tech/go-smapiweb/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/Rethunk-Tech/go-smapiweb/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/Rethunk-Tech/go-smapiweb/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/Rethunk-Tech/go-smapiweb/releases/tag/v0.1.0
