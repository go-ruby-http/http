// Copyright (c) the go-ruby-http/http authors
//
// SPDX-License-Identifier: BSD-3-Clause

package http

// This file is the top-level "HTTP." surface: the package-level chainable
// entry points and verb shortcuts that mirror Ruby's HTTP.get / HTTP.auth /
// HTTP.headers … which delegate to a fresh default [Client]. A chain that must
// start by setting headers uses NewClient().Headers(...) (the package-level name
// Headers is taken by the [Headers] type).

// Get issues a GET request with a fresh default client (HTTP.get).
func Get(uri string, opts ...RequestOption) (*Response, error) {
	return NewClient().Get(uri, opts...)
}

// Head issues a HEAD request with a fresh default client (HTTP.head).
func Head(uri string, opts ...RequestOption) (*Response, error) {
	return NewClient().Head(uri, opts...)
}

// Post issues a POST request with a fresh default client (HTTP.post).
func Post(uri string, opts ...RequestOption) (*Response, error) {
	return NewClient().Post(uri, opts...)
}

// Put issues a PUT request with a fresh default client (HTTP.put).
func Put(uri string, opts ...RequestOption) (*Response, error) {
	return NewClient().Put(uri, opts...)
}

// Delete issues a DELETE request with a fresh default client (HTTP.delete).
func Delete(uri string, opts ...RequestOption) (*Response, error) {
	return NewClient().Delete(uri, opts...)
}

// Patch issues a PATCH request with a fresh default client (HTTP.patch).
func Patch(uri string, opts ...RequestOption) (*Response, error) {
	return NewClient().Patch(uri, opts...)
}

// Header starts a chain by setting one default header (HTTP.headers(k => v)).
func Header(key, val string) *Client { return NewClient().Header(key, val) }

// Auth starts a chain by setting the Authorization header (HTTP.auth).
func Auth(value string) *Client { return NewClient().Auth(value) }

// BasicAuth starts a chain with HTTP Basic Authorization (HTTP.basic_auth).
func BasicAuth(user, pass string) *Client { return NewClient().BasicAuth(user, pass) }

// Accept starts a chain by setting the Accept header (HTTP.accept).
func Accept(typ string) *Client { return NewClient().Accept(typ) }

// Timeout starts a chain with a global timeout (HTTP.timeout).
func Timeout(seconds int) *Client { return NewClient().Timeout(seconds) }

// Follow starts a chain that follows redirects (HTTP.follow).
func Follow(opts ...FollowOptions) *Client { return NewClient().Follow(opts...) }

// Via starts a chain routing through a proxy (HTTP.via).
func Via(host string, port int, userpass ...string) *Client {
	return NewClient().Via(host, port, userpass...)
}

// Through is an alias for [Via] (HTTP.through).
func Through(host string, port int, userpass ...string) *Client {
	return NewClient().Through(host, port, userpass...)
}

// Cookies starts a chain with default cookies (HTTP.cookies).
func Cookies(kv ...KV) *Client { return NewClient().Cookies(kv...) }

// Encoding starts a chain forcing the response charset (HTTP.encoding).
func Encoding(enc string) *Client { return NewClient().Encoding(enc) }

// Nodelay starts a chain with TCP_NODELAY (HTTP.nodelay).
func Nodelay() *Client { return NewClient().Nodelay() }

// Use starts a chain with features enabled (HTTP.use).
func Use(features ...string) *Client { return NewClient().Use(features...) }

// Persistent runs block against a persistent client pinned to host
// (HTTP.persistent(host) { |http| ... }).
func Persistent(host string, block func(*Client) error) error {
	return NewClient().Persistent(host, block)
}
