<h1 align="center">go-smapiweb</h1>

<div align="center">

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.27.3+-00ADD8?logo=go&logoColor=white)](go.mod)

</div>

---

A client for the web services of the Stardew Valley modding toolchain: smapi.io's mod update API and log parser, the SMAPI compatibility list and the StardewModDataset. Package `smapiweb`, module `github.com/Rethunk-Tech/go-smapiweb`.

The services publish no client for Go and answer in shapes that drift. This package sends the requests, bounds every
body it reads, and parses leniently. It caches nothing: the `Updates` result says which mods were answered, so a
caller keeps its own cache and decides when an answer is too old. The JSON tags on the result and dataset types are a
format contract, so cached copies survive an upgrade.

## Quick start

```bash
go get github.com/Rethunk-Tech/go-smapiweb
```

A worked example and the contracts: [HUMANS.md](HUMANS.md).

## Features

- `Updates`: newest and unofficial versions and compatibility for any number of mods, batched, with per-mod `Known`.
- `CompatList`: the SMAPI compatibility list in every shape it has been published in, with `Lookup` and `BrokenOn`.
- `Index` and `Page`: which Nexus pages ship a UniqueID, and a page's files, manifests and dependencies.
- `UploadLog`: share a SMAPI log through smapi.io and get its public link.
- Every response is read under a size cap and a deadline.

## Documentation

| Doc | Audience |
| --- | --- |
| [HUMANS.md](HUMANS.md) | Running and using the package |
| [AGENTS.md](AGENTS.md) | Internals, invariants, and why |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Process — commits, tests, docs |
| [SECURITY.md](SECURITY.md) | Vulnerability disclosure |
| [CHANGELOG.md](CHANGELOG.md) | Release notes |

## License

MIT — see [LICENSE](LICENSE). © 2026 Rethunk.Tech, LLC.
