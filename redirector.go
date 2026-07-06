// Copyright (c) the go-ruby-http/http authors
//
// SPDX-License-Identifier: BSD-3-Clause

package http

import "net/url"

// redirectCodes is HTTP::Redirector::REDIRECT_CODES.
var redirectCodes = map[int]bool{300: true, 301: true, 302: true, 303: true, 307: true, 308: true}

// strictSensitiveCodes is HTTP::Redirector::STRICT_SENSITIVE_CODES.
var strictSensitiveCodes = map[int]bool{300: true, 301: true, 302: true}

// unsafeVerbs is HTTP::Redirector::UNSAFE_VERBS.
var unsafeVerbs = map[string]bool{"PUT": true, "DELETE": true, "POST": true}

// seeOtherAllowedVerbs is HTTP::Redirector::SEE_OTHER_ALLOWED_VERBS.
var seeOtherAllowedVerbs = map[string]bool{"GET": true, "HEAD": true}

// followRedirects re-issues a request through perform until a non-redirect
// response is reached, mirroring HTTP::Redirector#perform. It enforces the hop
// limit ([FollowOptions.MaxHops], defaulting to [DefaultMaxRedirects]), detects
// an endless loop, and applies http.rb's verb/header redirect policy
// ([FollowOptions.Strict]).
func followRedirects(perform func(*Request) (*Response, error), req *Request, resp *Response, follow *FollowOptions) (*Response, error) {
	maxHops := follow.MaxHops
	if maxHops == 0 {
		maxHops = DefaultMaxRedirects
	}
	var visited []string
	for redirectCodes[resp.Code()] {
		key := visitKey(req)
		visited = append(visited, key)
		if maxHops > 0 && len(visited) > maxHops {
			return nil, newError(KindResponseError, "too many redirects")
		}
		if countString(visited, key) > 1 {
			return nil, newError(KindResponseError, "endless redirect detected")
		}
		next, err := redirectTo(req, resp, follow.Strict)
		if err != nil {
			return nil, err
		}
		req = next
		if resp, err = perform(req); err != nil {
			return nil, err
		}
	}
	return resp, nil
}

// redirectTo builds the next request for a redirect response, mirroring
// HTTP::Redirector#redirect_to and HTTP::Request#redirect_headers: it resolves
// the Location, downgrades/keeps the verb per policy, strips the Host header,
// strips Authorization/Cookie on a cross-origin hop, and strips Content-Type
// (and the body) when the verb becomes GET.
func redirectTo(req *Request, resp *Response, strict bool) (*Request, error) {
	loc, ok := resp.Headers().Get("Location")
	if !ok || loc == "" {
		return nil, newError(KindStateError, "no Location header in redirect")
	}
	verb := req.Verb
	code := resp.Code()

	if unsafeVerbs[verb] && strictSensitiveCodes[code] {
		if strict {
			return nil, newError(KindStateError, "can't follow "+resp.Status().String()+" redirect")
		}
		verb = "GET"
	}
	if !seeOtherAllowedVerbs[verb] && code == 303 {
		verb = "GET"
	}

	target := resolveURL(req.URL, loc)

	h := req.Headers.Clone()
	h.Delete("Host")
	if !sameOrigin(req.URL, target) {
		h.Delete("Authorization")
		h.Delete("Cookie")
	}
	body := req.Body
	if verb == "GET" {
		h.Delete("Content-Type")
		body = ""
	}
	return &Request{Verb: verb, URL: target, Headers: h, Body: body, Cookies: req.Cookies, Proxy: req.Proxy}, nil
}

// visitKey is HTTP::Redirector#visit_key: "VERB URL Cookie", so the same URL
// with different cookies is not a false loop.
func visitKey(req *Request) string {
	cookie, _ := req.Headers.Get("Cookie")
	return req.Verb + " " + req.URL + " " + cookie
}

// resolveURL resolves a (possibly relative) Location against the current request
// URL. An unparsable base or location falls back to the raw location string.
func resolveURL(base, loc string) string {
	b, err := url.Parse(base)
	if err != nil {
		return loc
	}
	ref, err := url.Parse(loc)
	if err != nil {
		return loc
	}
	return b.ResolveReference(ref).String()
}

// sameOrigin reports whether two URLs share scheme+host+port (an unparsable URL
// is treated as a different origin, so credentials are stripped defensively).
func sameOrigin(a, b string) bool {
	ua, err := url.Parse(a)
	if err != nil {
		return false
	}
	ub, err := url.Parse(b)
	if err != nil {
		return false
	}
	return ua.Scheme == ub.Scheme && ua.Host == ub.Host
}

// countString returns how many times s appears in xs.
func countString(xs []string, s string) int {
	n := 0
	for _, x := range xs {
		if x == s {
			n++
		}
	}
	return n
}
