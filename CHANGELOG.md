# Changelog

Notable changes to `go-smapiweb`. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `Client.Index` and `Client.Page` read the StardewModDataset: which Nexus pages ship a UniqueID, and a page's files, manifests and dependencies. A page the dataset lacks is an empty `Page`.
- `Client.CompatList` reads the SMAPI compatibility list in every shape it has been published in, into a `CompatIndex` with `Lookup` by UniqueID or Nexus page and `CompatEntry.BrokenOn` for a game version.

[Unreleased]: https://github.com/Rethunk-Tech/go-smapiweb/commits/main
