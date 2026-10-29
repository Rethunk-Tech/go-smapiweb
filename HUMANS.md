# HUMANS.md

Run and use `go-smapiweb`. Internals: [AGENTS.md](AGENTS.md).

## Quick start

```bash
go get github.com/Rethunk-Tech/go-smapiweb
```

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Rethunk-Tech/go-smapiweb"
)

func main() {
	var c smapiweb.Client // the zero value uses the public endpoints

	res, err := c.Updates(context.Background(), smapiweb.UpdateRequest{
		GameVersion: "1.6.15",
		Platform:    "Linux",
		Mods: []smapiweb.InstalledMod{
			{ID: "Pathoschild.ContentPatcher", UpdateKeys: []string{"Nexus:1915"}, Version: "2.0.0"},
		},
	})
	if err != nil { // some batches failed; the results still hold the answered ones
		log.Print(err)
	}
	for _, r := range res {
		switch {
		case !r.Known:
			fmt.Println(r.ID, "unknown")
		case r.Suggested != nil:
			fmt.Println(r.ID, "can update to", r.Suggested.Version)
		default:
			fmt.Println(r.ID, "is current")
		}
	}

	list, err := c.CompatList(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	if e, ok := list.Lookup("Pathoschild.ContentPatcher", 1915); ok && e.BrokenOn("1.6.15") {
		fmt.Println("broken:", e.Summary)
	}
}
```

`Index`, `Page` and `UploadLog` take the same `Client`. Point a field at an `httptest` server to test against
no real service.

## Contracts

| Call | Contract |
| --- | --- |
| `Updates` | One result per mod, in request order, with the request's ID. `Known` is true only for a mod whose batch was answered; a mod the API does not list is `Known` with no `Suggested`. The error is `errors.Join` of the failed batches' errors in batch order; it is nil only when every batch was answered. |
| `Updates` batching | 100 mods a batch, four at a time; a failed batch is asked again in two halves; after a failed batch, batches not yet sent are not sent. |
| `CompatList` | Parses every published shape of the list; a status the list leaves out is inferred; a malformed entry is skipped, a malformed document is an error. |
| `CompatIndex.Lookup` | UniqueID (any case) first, then Nexus page id. |
| `CompatEntry.BrokenOn` | A break in a later game version is not a break; an unparseable version is. |
| `Index` | Keys are lowercased UniqueIDs; an index with no usable entry is an error. |
| `Page` | A page the dataset lacks is an empty `Page` and a nil error. A file name that is a storage path is blanked. |
| `UploadLog` | Sends nothing for empty text or text over `maxBytes` (0 means no cap). Only a redirect to `/log/<id>` is success: 200 is `ErrLogRejected`, another status a `*httpkit.StatusError`. Never modifies `Client.HTTP`. |
| Errors | Transport failures stay `*url.Error`; a status other than 200 is a `*httpkit.StatusError` (`errors.Is(err, httpkit.ErrNotFound)` works); a body over its cap is `safefs.ErrTooLarge`. |
| State | None: no cache, no logging, no backoff across calls. |
| JSON | The tags on the result and dataset types are a format contract, safe to cache. |

## Requirements

Go 1.27.3 or newer, plus `lefthook` and `gitleaks` on PATH if you want the git hooks.

```bash
lefthook install
```

That is per clone; git does not commit hooks. Pre-commit scans staged changes for secrets, and
pre-push runs `gate`.

## Verify

```bash
gate
```
