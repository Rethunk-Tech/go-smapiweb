package smapiweb

import (
	"context"
	"net/http"
	"time"

	"github.com/Rethunk-Tech/go-httpkit"
)

// requestTimeout bounds one ask of a read endpoint; smapi.io answers a full batch in about a second when it is
// healthy, so a longer wait is a stuck connection.
const requestTimeout = 20 * time.Second

// Client talks to smapi.io and the two GitHub-hosted datasets. The zero value uses the public endpoints and
// http.DefaultClient. It holds no state, so one Client is safe for concurrent use and a copy is as good as the
// original.
type Client struct {
	// HTTP is the client every request goes through; nil means http.DefaultClient. UploadLog adds httpkit.NoFollow
	// on a copy, so the client is never modified.
	HTTP *http.Client
	// UpdatesURL is the mod update API; "" means smapi.io's v4.0.0 endpoint.
	UpdatesURL string
	// CompatURL is the compatibility list; "" means SmapiCompatibilityList's data/mods.jsonc on its develop branch.
	CompatURL string
	// IndexURL is the dataset's page-by-UniqueID index; "" means StardewModDataset's on its main branch.
	IndexURL string
	// PageBase is the folder holding the dataset's Nexus pages; "" means StardewModDataset's on its main branch.
	PageBase string
	// LogURL is the log parser; "" means smapi.io's.
	LogURL string
}

// do sends req and returns at most limit bytes of a 200 response body, under requestTimeout. check is nil: none of
// these hosts sends anything worth inspecting before the status test.
func (c *Client) do(req *http.Request, limit int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(req.Context(), requestTimeout)
	defer cancel()
	return httpkit.DoCapped(c.HTTP, req.WithContext(ctx), limit, nil)
}
