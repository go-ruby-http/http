// Copyright (c) the go-ruby-http/http authors
//
// SPDX-License-Identifier: BSD-3-Clause

package http

// Options is the immutable per-client configuration, mirroring
// HTTP::Options (a client's default_options). The chainable methods on [Client]
// branch a client by cloning its Options and tweaking one field, exactly as
// http.rb's HTTP::Chainable#branch does.
type Options struct {
	// Headers are the default headers applied to every request.
	Headers *Headers
	// Cookies are the default cookies sent with every request.
	Cookies *CookieJar
	// Proxy is the upstream proxy (nil ⇒ direct), set by Via.
	Proxy *Proxy
	// Follow configures redirect following (nil ⇒ do not follow).
	Follow *FollowOptions
	// Timeout carries the request timeouts (nil ⇒ unset), set by Timeout.
	Timeout *TimeoutOptions
	// Persistent is the host a persistent client is pinned to ("" ⇒ per-request).
	Persistent string
	// Features are the enabled feature/middleware names (HTTP#use).
	Features []string
	// Encoding forces the response body charset (HTTP#encoding).
	Encoding string
	// Nodelay disables Nagle's algorithm (HTTP#nodelay); host-honored metadata.
	Nodelay bool
}

// Proxy is an upstream HTTP proxy, mirroring the `proxy` option built by
// HTTP::Chainable#via.
type Proxy struct {
	Host string
	Port int
	User string
	Pass string
}

// TimeoutOptions carries http.rb's timeout configuration. Mode is "" for the
// default (global) timeout or "per_operation" when connect/read/write are set
// individually. Values are in seconds; 0 means unset. The deterministic core
// does not open sockets, so these are metadata a host transport honors.
type TimeoutOptions struct {
	Mode    string
	Global  int
	Connect int
	Read    int
	Write   int
}

// FollowOptions configures redirect following, mirroring the `follow` option and
// HTTP::Redirector. MaxHops caps the redirect chain (default
// [DefaultMaxRedirects]); Strict keeps the original verb on 301/302/307/308
// (http.rb's default), while a non-strict follow downgrades 301/302 to GET.
type FollowOptions struct {
	MaxHops int
	Strict  bool
}

// DefaultMaxRedirects is HTTP::Redirector's default max_hops.
const DefaultMaxRedirects = 5

// clone returns a deep copy of o so a branched client never mutates its parent's
// configuration.
func (o *Options) clone() *Options {
	c := *o
	if o.Headers != nil {
		c.Headers = o.Headers.Clone()
	}
	if o.Cookies != nil {
		c.Cookies = o.Cookies.Clone()
	}
	if o.Proxy != nil {
		p := *o.Proxy
		c.Proxy = &p
	}
	if o.Follow != nil {
		f := *o.Follow
		c.Follow = &f
	}
	if o.Timeout != nil {
		t := *o.Timeout
		c.Timeout = &t
	}
	if o.Features != nil {
		c.Features = append([]string(nil), o.Features...)
	}
	return &c
}

// headers returns o.Headers, initializing an empty set if needed (never nil).
func (o *Options) ensureHeaders() *Headers {
	if o.Headers == nil {
		o.Headers = NewHeaders()
	}
	return o.Headers
}
