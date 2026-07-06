// Copyright (c) the go-ruby-http/http authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package http is a pure-Go (CGO-free) reimplementation of the deterministic
// core of Ruby's `http` gem — the chainable "http.rb" HTTP client
// (HTTP.get/post/…), NOT Ruby's stdlib net/http. It reproduces the chainable
// client DSL, the request/response objects, the status/headers/content-type/
// cookie value model, the form/JSON/params body encodings, redirect following,
// and the HTTP::Error tree that http.rb raises — without any Ruby runtime.
//
// # What it is — and isn't
//
// Everything http.rb does *around* the wire is deterministic and needs no
// interpreter, so it lives here as pure Go: branching the chainable client
// (headers/auth/basic_auth/accept/timeout/follow/via/persistent), building the
// request (URL, merged headers, form/JSON/raw/params body encoding), following
// redirects, and exposing the [Response] (status predicates, headers,
// content-type, JSON parse, cookies). The HTTP round-trip itself is a host
// seam: the [Transport] performs the transport. The default production
// Transport is [NetTransport], backed by net/http; tests inject a
// [TransportFunc] stub, and the core opens no socket itself.
//
// # Flow
//
//	resp, err := http.NewClient().
//		Headers(http.KV{Key: "X-Trace", Val: "1"}).
//		Auth("Bearer tok").
//		Accept("json").
//		Timeout(30).
//		Follow().
//		Get("https://api.example.com/widgets")
//	if err != nil { /* an *http.Error: TimeoutError, ConnectionError, … */ }
//	_ = resp.Status().Success()   // true for 2xx
//	_ = resp.Body().String()      // raw body
//	v, _ := resp.Parse()          // parsed JSON (by Content-Type)
//
// A POST with a body encoding:
//
//	resp, err := http.Post("https://api.example.com/widgets",
//		http.JSON(map[string]any{"name": "gadget"}))
//
// # Value model
//
// A host (go-embedded-ruby / rbgo) maps its Ruby HTTP::Client / HTTP::Request /
// HTTP::Response / HTTP::Headers / HTTP::Response::Status objects to and from
// these Go shapes, and supplies the production [Transport].
package http
