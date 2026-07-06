<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-http/brand/main/social/go-ruby-http-http.png" alt="go-ruby-http/http" width="720"></p>

# http — go-ruby-http

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-http.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of the deterministic core of Ruby's
[`http`](https://github.com/httprb/http) gem — "http.rb"** — the chainable
`HTTP.get(...)` client (**not** Ruby's stdlib `net/http`). It reproduces the
chainable client DSL, the request/response objects, the
status / headers / content-type / cookie value model, the form / JSON / params
body encodings, redirect following, and the `HTTP::Error` tree — **without any
Ruby runtime**.

It is the http.rb client for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but a **standalone,
reusable** module — a sibling of
[go-ruby-faraday](https://github.com/go-ruby-faraday/faraday),
[go-ruby-regexp](https://github.com/go-ruby-regexp/regexp) and
[go-ruby-erb](https://github.com/go-ruby-erb/erb).

> **What it is — and isn't.** Everything http.rb does *around* the wire is
> deterministic and needs **no interpreter**, so it lives here as pure Go:
> branching the chainable client (headers / auth / accept / timeout / follow /
> via / persistent), building the request (URL, merged headers, form/JSON/raw/
> params body encoding), following redirects, and exposing the `Response`
> (status predicates, headers, content-type, JSON parse, cookies). The **HTTP
> round-trip itself is a host seam**: the `Transport` performs the transport. The
> default production `Transport` is `NetTransport`, backed by `net/http`; **tests
> inject a `TransportFunc` stub and the core opens no socket itself.**

## Features

Faithful port of the `http` gem's client core, validated **byte-for-byte**
against the gem (MRI) on every platform where it is installed:

- **Chainable client** — package-level `http.Get/Post/Put/Delete/Head/Patch` and
  the builder chain
  `http.NewClient().Headers(...).Auth(...).BasicAuth(u,p).Accept("json").Timeout(30).Follow().Via(host,port).Get(url)`,
  every link returning a new branched `*Client` (immutable, like http.rb).
- **Verb methods** — `Get`/`Head`/`Post`/`Put`/`Delete`/`Patch`/`Options`/`Trace`/`Connect`
  and the general `Request(verb, uri, opts...)`.
- **Body encodings** — request options `Form(*Values)` (`a=1&b=x+y&c[]=1&c[]=2`),
  `JSON(any)` (`application/json; charset=utf-8`), `Body(string)` (raw), and
  `Params(*Values)` (merged into the query string). Precedence body ▸ form ▸ json,
  matching the gem; a `GET` strips `Content-Type`.
- **Response** — `Status()` (a `Status` with `.Success()`/`.Redirect()`/
  `.ClientError()`/`.ServerError()`/`.Informational()`, `.Reason()`, `.Code()`),
  `Body().String()`, `Headers()`, `ContentType()`, `Parse()` (JSON by
  Content-Type, or an override), and `Cookies()`.
- **Persistent connections** — `http.Persistent(host) { |c| ... }` pins a client
  to one host; a request to another host raises a `StateError`.
- **Redirect following** — `Follow()` mirrors `HTTP::Redirector`: max-hops cap,
  endless-loop detection, strict/non-strict verb downgrade (301/302/303),
  cross-origin credential stripping.
- **Error tree** — `HTTP::Error` → `ConnectionError`, `RequestError`,
  `ResponseError` (→ `StateError`), `TimeoutError`, `HeaderError`, matched with
  `errors.Is` against the `Err*` sentinels (a superclass matches its subclasses).
- **Transport seam** — `client.WithTransport(Transport)`; `DefaultTransport()` is
  the net/http transport, a `TransportFunc` a test stub. **The core never opens a
  socket.** A host wires `DefaultClientTransport` once for the package-level
  shortcuts.

CGO-free, dependency-free (stdlib only), **100% test coverage**, `gofmt` +
`go vet` clean, and green across the six 64-bit Go targets (amd64, arm64,
riscv64, loong64, ppc64le, **s390x** — big-endian).

## Install

```sh
go get github.com/go-ruby-http/http
```

## Usage

```go
package main

import (
	"errors"
	"fmt"

	"github.com/go-ruby-http/http"
)

func main() {
	resp, err := http.NewClient().
		Headers(http.KV{Key: "X-Trace", Val: "1"}).
		Auth("Bearer tok").
		Accept("json").
		Timeout(30).
		Follow().
		Get("https://api.example.com/widgets")
	if err != nil {
		// a *http.Error: errors.Is(err, http.ErrTimeoutError), etc.
		if errors.Is(err, http.ErrConnectionError) {
			return
		}
		return
	}
	fmt.Println(resp.Status().Code(), resp.Status().Success())
	fmt.Println(resp.Body().String())

	if resp.Status().Success() {
		v, _ := resp.Parse() // JSON parsed by Content-Type
		fmt.Println(v)
	}
}
```

### POST body encodings

```go
http.Post("https://api.example.com/widgets", http.JSON(map[string]any{"name": "gadget"}))
http.Post("https://api.example.com/form", http.Form(http.NewValues(http.KV{"a", "1"})))
http.Post("https://api.example.com/raw", http.Body("raw bytes"))
http.Get("https://api.example.com/search", http.Params(http.NewValues(http.KV{"q", "go http"})))
```

### Injecting a transport (tests / hosts)

```go
c := http.NewClient().WithTransport(http.TransportFunc(func(req *http.Request) (*http.Response, error) {
	return http.NewResponse(200,
		http.NewHeaders(http.KV{"Content-Type", "application/json"}),
		`{"ok":true}`, "1.1", req.URL), nil
}))
resp, _ := c.Get("/ping")
v, _ := resp.Parse() // map[string]any{"ok": true}
```

## Value model

| gem                                       | this package                                        |
| ----------------------------------------- | --------------------------------------------------- |
| `HTTP.get/post/...(url, opts)`            | `http.Get/Post/...(url, opts...)`                   |
| `HTTP.headers(...).auth(...).get(url)`    | `http.NewClient().Headers(...).Auth(...).Get(url)`  |
| `HTTP.post(url, form:/json:/body:/params:)` | `http.Post(url, http.Form/JSON/Body/Params(...))` |
| `HTTP::Response#status` (`.success?` …)   | `(*Response).Status()` (`.Success()` …)             |
| `HTTP::Response#body.to_s` / `#parse`     | `(*Response).Body().String()` / `.Parse()`          |
| `HTTP::Response#content_type` / `#cookies`| `(*Response).ContentType()` / `.Cookies()`          |
| `HTTP.persistent(host) { ... }`           | `http.Persistent(host, func(c *Client) error {...})`|
| `HTTP.follow` (`HTTP::Redirector`)        | `client.Follow()`                                   |
| `HTTP::Error` subtree                     | `*Error` + `Err*` sentinels (`errors.Is`)           |
| `HTTP::ContentType.parse`                 | `ParseContentType`                                  |
| `HTTP::Headers::Normalizer`               | `NormalizeHeaderName`                               |
| `HTTP::FormData::Urlencoded`              | `(*Values).Encode` / `EscapeFormComponent`          |

## Tests & coverage

The suite pairs deterministic, ruby-free tests (which alone hold coverage at
**100%**, so the qemu cross-arch and Windows lanes pass the gate) with a
**differential oracle** against the reference `http` gem: header-name
normalization, url-encoded form bodies, `URI.encode_www_form_component` escaping,
content-type parsing, status reason phrases / `to_s`, and Basic-auth header
construction are diffed **byte-for-byte** against the gem. The oracle scripts
`$stdout.binmode` and skip themselves where the gem is absent. **No test opens a
socket** — the transport is stubbed everywhere.

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-http/http authors.
