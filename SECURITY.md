# Security

## Supported versions

The latest tagged release. There are no long-lived release branches, so a fix ships in the next tag
rather than being backported.

## Reporting a vulnerability

Use GitHub's **private vulnerability reporting** on this repository (Security → Report a
vulnerability). It stays private until a fix ships.

Please do not open a public issue for a suspected vulnerability first.

## What is in scope

Every answer from smapi.io and raw.githubusercontent.com is untrusted, and so is whoever sits between.

- `Updates`, `CompatList`, `Index` and `Page` read a response body: it must never be read past its cap (16, 16, 32
  and 8 MiB), and a crafted document must not panic a parser or put an empty key in a result. Fuzz targets cover each
  parser.
- `Page` must never return a file name that holds a path separator, so a caller can use it as a file name.
- `UploadLog` follows no redirect and rebuilds the link from the redirect's scheme and host with the id
  path-escaped and slash-free, so a hostile `Location` cannot return a link to another path or inject one.
- Update answers must never alter the request's mod IDs: results carry the caller's IDs.

## What is out of scope

- Trust in the services themselves: a hostile smapi.io can say a mod has an update, with any URL. Callers decide
  whether to follow the `Update.URL` and `UnofficialURL` they get back.
- The endpoints a caller configures on `Client` are trusted; there is no allowlist.
- TLS and proxies are the `http.Client`'s business.
