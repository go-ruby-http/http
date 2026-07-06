// Copyright (c) the go-ruby-http/http authors
//
// SPDX-License-Identifier: BSD-3-Clause

package http

import (
	"encoding/json"
	"strings"
)

// Request is a prepared HTTP request handed to a [Transport], mirroring
// HTTP::Request: the verb, the target URL, the fully-merged headers (defaults
// overlaid by per-request headers and any encoded body's Content-Type), the
// encoded body, and the cookies/proxy the client is configured with.
type Request struct {
	// Verb is the upper-case HTTP method ("GET", "POST", …).
	Verb string
	// URL is the fully-built request URL (with any `params:` query merged in).
	URL string
	// Headers are the outgoing headers.
	Headers *Headers
	// Body is the encoded request body (form/JSON/raw); "" when there is none.
	Body string
	// Cookies are the cookies to send (rendered into the Cookie header by the
	// transport or already merged by the client).
	Cookies *CookieJar
	// Proxy is the upstream proxy, when configured.
	Proxy *Proxy
}

// RequestOption configures a single request body/query, mirroring the keyword
// arguments to http.rb's verb methods (`form:`, `json:`, `body:`, `params:`).
type RequestOption func(*reqBuild)

// reqBuild accumulates the per-request body/query choices before encoding.
type reqBuild struct {
	form    *Values
	jsonVal any
	hasJSON bool
	body    *string
	params  *Values
}

// Form sends v as an application/x-www-form-urlencoded body (http.rb `form:`).
func Form(v *Values) RequestOption { return func(b *reqBuild) { b.form = v } }

// JSON sends v JSON-encoded with Content-Type application/json; charset=utf-8
// (http.rb `json:`).
func JSON(v any) RequestOption {
	return func(b *reqBuild) { b.jsonVal = v; b.hasJSON = true }
}

// Body sends s as the raw request body verbatim (http.rb `body:`).
func Body(s string) RequestOption { return func(b *reqBuild) { b.body = &s } }

// Params merges v into the request URL's query string (http.rb `params:`).
func Params(v *Values) RequestOption { return func(b *reqBuild) { b.params = v } }

// buildRequest assembles a [Request] for verb/uri under opts (the client's
// default options) and the per-request choices, mirroring
// HTTP::Request::Builder#build: it merges default headers/cookies, applies the
// `params:` query, encodes the body (body ▸ form ▸ json precedence) and sets the
// Content-Type unless already present, and strips Content-Type on a GET.
func buildRequest(verb, uri string, opts *Options, ropts []RequestOption) *Request {
	var rb reqBuild
	for _, o := range ropts {
		o(&rb)
	}

	headers := NewHeaders()
	if opts.Headers != nil {
		headers = opts.Headers.Clone()
	}

	fullURL := mergeQuery(uri, rb.params)

	body := ""
	switch {
	case rb.body != nil:
		body = *rb.body
	case rb.form != nil:
		body = rb.form.Encode()
		headers.SetDefault("Content-Type", "application/x-www-form-urlencoded")
	case rb.hasJSON:
		data, _ := json.Marshal(rb.jsonVal)
		body = string(data)
		headers.SetDefault("Content-Type", "application/json; charset=utf-8")
	}

	up := strings.ToUpper(verb)
	if up == "GET" {
		headers.Delete("Content-Type")
	}

	var jar *CookieJar
	if opts.Cookies != nil {
		jar = opts.Cookies.Clone()
		if v := jar.HeaderValue(); v != "" {
			headers.SetDefault("Cookie", v)
		}
	}

	return &Request{
		Verb:    up,
		URL:     fullURL,
		Headers: headers,
		Body:    body,
		Cookies: jar,
		Proxy:   opts.Proxy,
	}
}

// mergeQuery appends the encoded params to uri's query string (with '?' or '&'),
// mirroring the way HTTP::Request::Builder concatenates `params:` onto the URL's
// existing query. A nil or empty params leaves uri unchanged.
func mergeQuery(uri string, params *Values) string {
	if params == nil || params.Len() == 0 {
		return uri
	}
	enc := params.Encode()
	if strings.ContainsRune(uri, '?') {
		return uri + "&" + enc
	}
	return uri + "?" + enc
}
