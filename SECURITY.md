# Security

## Supported versions

The latest tagged release. There are no long-lived release branches, so a fix ships in the next tag
rather than being backported.

## Reporting a vulnerability

Use GitHub's private vulnerability reporting on this repository: <https://github.com/Rethunk-Tech/go-smapiweb/security/advisories/new>, or the **Security** tab, then **Report a vulnerability**. The report stays private between you and us until a fix ships.

If private reporting is unavailable, email [oss@rethunk.tech](mailto:oss@rethunk.tech) with **[SECURITY]** in the subject.

Please do not open a public issue, pull request or discussion for a suspected vulnerability.

A useful report has the affected repository and version (or commit), steps to reproduce, what an attacker gets and under which conditions, and whether anyone else knows.

## What counts as a vulnerability

A vulnerability is a flaw that lets someone cross a boundary they should not: run code, read or change data they have no right to, escape a sandbox or path restriction, or leak a credential. A bug is anything else that misbehaves, such as a crash on your own input, wrong output or a slow path; open an ordinary issue for those. If you are unsure which one you have, report it privately and we will say.

## Coordinated disclosure

We handle the report inside the private advisory, so you see each status change and can comment on the fix. The advisory goes public when the fix ships, or 90 days after your report, whichever comes first. If a fix is close, either of us can ask for more time in the thread. You are credited in the advisory unless you would rather not be.

## Response timelines

- We acknowledge your report within 3 business days.
- We give you an assessment (accepted, needs more information or declined, with reasons) within 10 business days.
- We aim to ship a fix within 90 days of the report.

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

## Safe harbour

We will not pursue legal action against good-faith research that avoids privacy violations, data destruction and disruption of services, stops at a proof of concept, and keeps the details private until the advisory is published. Rethunk.Tech, LLC maintains this repository.
