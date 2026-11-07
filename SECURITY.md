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
- `Page` must never return a file name that is `.` or `..` or holds a path separator, a colon or a control character, so a caller can use it as a file name.
- `Index` keeps at most 64 distinct pages per UniqueID, so a hostile index cannot multiply its size in memory.
- `UploadLog` follows no redirect and rebuilds the link from the redirect's scheme and host with the id
  path-escaped and slash-free; a `Location` on another scheme or host than the request is `ErrNoLogLink`.
- A 307 or 308 to another origin is refused, since it would re-send the installed-mod list; an https-to-http redirect is refused for every read.
- `Update.URL` and `UnofficialURL` are http or https addresses or empty.
- Update answers must never alter the request's mod IDs: results carry the caller's IDs.

## What is out of scope

- Trust in the services themselves: a hostile smapi.io can say a mod has an update, with any URL. Callers decide
  whether to follow the http(s) `Update.URL` and `UnofficialURL` they get back.
- The endpoints a caller configures on `Client` are trusted; there is no allowlist.
- TLS and proxies are the `http.Client`'s business.
