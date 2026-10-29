# AGENTS.md

`smapiweb` is a client for four public Stardew Valley modding services: smapi.io's update API and log parser, the SMAPI
compatibility list and the StardewModDataset. Any Go program around Stardew can import it. It imports no consumer
package, no UI framework and no data-folder code, and it holds no disk state.

## Gate

`gate` is the command. Each override in `.gate.toml` is there because a runner
cannot resolve that check from PATH.

## Shape

- **Stateless.** No cache, TTL, build stamp, cross-call backoff or logging. Staleness is policy that changes often and
  differs per consumer; `Updates` hands it what it needs instead (`Known` per mod, the joined error).
- **`Updates` is the only batcher.** Batches of 100, four in flight, a failed batch asked once more in halves unless
  the context is done or it holds one mod, and no batch sent after one failed. The stop is call-local: nothing
  outlives the call. The joined error holds only real failures, never an unsent batch, so its first element is
  always a transport or status error.
- **Every read goes through `Client.do`**: a 20 s deadline and `httpkit.DoCapped`, so a body over its cap is an error
  rather than an allocation. The caps (16 MiB update answer and compatibility list, 32 MiB index, 8 MiB page) are about
  17 times the live sizes and stay unexported; a caller wanting a shorter deadline passes a context.
- **Parsers are lenient on purpose.** The upstream files are hand-edited or written by other tools: a field of the
  wrong type keeps the rest of the record, and `parse.go`, `entryFromRaw` and `decodeUpdateAnswers` never panic.
  Fuzz targets assert that.
- **JSON tags are API.** `UpdateResult`, `Update`, `CompatEntry`, `CompatIndex` (field-name keys), `Ref`, `Page`,
  `File`, `Mod` and `Dependency` are cached by callers, some without a stamp, so a rename reads back as zero values
  with no error. `TestJSONShapeIsStable` pins them.
- **A 404 page is an empty `Page`**: the dataset answered, it just lacks the page. Any other status is an error.
- **`UploadLog` success is a redirect to `/log/<id>`**; smapi.io answers a refusal with 200 and its page. The link is
  rebuilt from the redirect's scheme and host, the id path-escaped. It adds `httpkit.NoFollow` to a copy of the
  client, never the caller's.
- **No host name in an error from `do` or `UploadLog`'s status error**: `httpkit.StatusError` words them, and a
  caller adds its own wording.

## Files

| File | Holds |
| --- | --- |
| `client.go` | `Client`, the endpoint fields and `do`, the single capped, time-bounded read path |
| `updates.go` | `Updates`, its batching and retry, the update API's request and answer types |
| `compat.go` | `CompatList`, `CompatIndex`, `CompatEntry`, `BrokenOn`, and the parser for the list's schema shapes |
| `dataset.go` | `Index`, `Page` and the dataset types; the index parser |
| `parse.go` | Lenient value types and the page parser |
| `log.go` | `UploadLog` and its errors |
