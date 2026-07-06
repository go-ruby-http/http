// Copyright (c) the go-ruby-http/http authors
//
// SPDX-License-Identifier: BSD-3-Clause

package http

import (
	"encoding/base64"
	"net/url"
)

// Client is an http.rb chainable client (HTTP::Client, including HTTP::Chainable):
// an immutable configuration ([Options]) plus a [Transport]. Each chainable
// method (Headers/Auth/Accept/…) returns a new branched client, so a base client
// can be shared and specialized without mutation, exactly like http.rb's
// chainable DSL. The verb methods issue requests through the transport.
type Client struct {
	options   *Options
	transport Transport
}

// DefaultClientTransport is the [Transport] a [NewClient] — and thus every
// package-level shortcut (Get/Post/Auth/…) — starts with. A host wires the
// production transport here once; tests swap in a stub so the package-level
// entry points never open a socket.
var DefaultClientTransport Transport = DefaultTransport()

// NewClient returns a fresh client with empty options and the
// [DefaultClientTransport]. Inject a per-client stub with [Client.WithTransport].
func NewClient() *Client {
	return &Client{options: &Options{}, transport: DefaultClientTransport}
}

// Options returns the client's current configuration (HTTP::Client#default_options).
func (c *Client) DefaultOptions() *Options { return c.options }

// WithTransport returns a branched client whose round-trips run through t — the
// host seam. Tests pass a [TransportFunc] stub; a host wires the real transport.
func (c *Client) WithTransport(t Transport) *Client {
	return &Client{options: c.options.clone(), transport: t}
}

// branch clones the client's options, applies mutate, and returns a new client
// sharing the transport, mirroring HTTP::Chainable#branch.
func (c *Client) branch(mutate func(*Options)) *Client {
	o := c.options.clone()
	mutate(o)
	return &Client{options: o, transport: c.transport}
}

// Headers returns a branched client with the given headers merged onto its
// defaults (HTTP.headers(...)).
func (c *Client) Headers(kv ...KV) *Client {
	return c.branch(func(o *Options) {
		h := o.ensureHeaders()
		for _, e := range kv {
			h.Set(e.Key, e.Val)
		}
	})
}

// Header returns a branched client with a single default header set.
func (c *Client) Header(key, val string) *Client { return c.Headers(KV{key, val}) }

// Auth returns a branched client with the Authorization header set to value
// verbatim (HTTP.auth("Bearer tok")).
func (c *Client) Auth(value string) *Client {
	return c.branch(func(o *Options) { o.ensureHeaders().Set("Authorization", value) })
}

// BasicAuth returns a branched client with HTTP Basic Authorization
// (HTTP.basic_auth(user:, pass:)): "Basic base64(user:pass)".
func (c *Client) BasicAuth(user, pass string) *Client {
	return c.Auth(BasicAuthHeader(user, pass))
}

// Accept returns a branched client with the Accept header set to the media type
// for typ (HTTP.accept(:json)). The alias "json"/":json" expands to
// "application/json"; any other value is used verbatim.
func (c *Client) Accept(typ string) *Client {
	return c.branch(func(o *Options) { o.ensureHeaders().Set("Accept", acceptType(typ)) })
}

// Timeout returns a branched client with a global timeout of seconds
// (HTTP.timeout(30)). The deterministic core does not open sockets; the value is
// metadata a host transport honors.
func (c *Client) Timeout(seconds int) *Client {
	return c.branch(func(o *Options) { o.Timeout = &TimeoutOptions{Global: seconds} })
}

// TimeoutOps returns a branched client with per-operation timeouts
// (HTTP.timeout(connect:, read:, write:)).
func (c *Client) TimeoutOps(connect, read, write int) *Client {
	return c.branch(func(o *Options) {
		o.Timeout = &TimeoutOptions{Mode: "per_operation", Connect: connect, Read: read, Write: write}
	})
}

// Follow returns a branched client that follows redirects (HTTP.follow). Pass a
// [FollowOptions] to override max hops / strictness; the default is strict with
// [DefaultMaxRedirects] hops.
func (c *Client) Follow(opts ...FollowOptions) *Client {
	return c.branch(func(o *Options) {
		f := FollowOptions{MaxHops: DefaultMaxRedirects, Strict: true}
		if len(opts) > 0 {
			f = opts[0]
		}
		o.Follow = &f
	})
}

// Via returns a branched client routing through an HTTP proxy
// (HTTP.via(host, port)); an optional user and password add proxy auth. The
// alias [Client.Through] matches http.rb's `through`.
func (c *Client) Via(host string, port int, userpass ...string) *Client {
	return c.branch(func(o *Options) {
		p := &Proxy{Host: host, Port: port}
		if len(userpass) > 0 {
			p.User = userpass[0]
		}
		if len(userpass) > 1 {
			p.Pass = userpass[1]
		}
		o.Proxy = p
	})
}

// Through is an alias for [Client.Via] (HTTP.through).
func (c *Client) Through(host string, port int, userpass ...string) *Client {
	return c.Via(host, port, userpass...)
}

// Cookies returns a branched client with the given default cookies added
// (HTTP.cookies(...)).
func (c *Client) Cookies(kv ...KV) *Client {
	return c.branch(func(o *Options) {
		if o.Cookies == nil {
			o.Cookies = NewCookieJar()
		}
		for _, e := range kv {
			o.Cookies.Add(e.Key, e.Val)
		}
	})
}

// Encoding returns a branched client forcing the response body charset
// (HTTP.encoding("UTF-8")).
func (c *Client) Encoding(enc string) *Client {
	return c.branch(func(o *Options) { o.Encoding = enc })
}

// Nodelay returns a branched client with TCP_NODELAY requested (HTTP.nodelay).
func (c *Client) Nodelay() *Client {
	return c.branch(func(o *Options) { o.Nodelay = true })
}

// Use returns a branched client with the named features/middleware enabled
// (HTTP.use(...)).
func (c *Client) Use(features ...string) *Client {
	return c.branch(func(o *Options) { o.Features = append(o.Features, features...) })
}

// Get issues a GET request (HTTP.get).
func (c *Client) Get(uri string, opts ...RequestOption) (*Response, error) {
	return c.Request("GET", uri, opts...)
}

// Head issues a HEAD request (HTTP.head).
func (c *Client) Head(uri string, opts ...RequestOption) (*Response, error) {
	return c.Request("HEAD", uri, opts...)
}

// Post issues a POST request (HTTP.post).
func (c *Client) Post(uri string, opts ...RequestOption) (*Response, error) {
	return c.Request("POST", uri, opts...)
}

// Put issues a PUT request (HTTP.put).
func (c *Client) Put(uri string, opts ...RequestOption) (*Response, error) {
	return c.Request("PUT", uri, opts...)
}

// Delete issues a DELETE request (HTTP.delete).
func (c *Client) Delete(uri string, opts ...RequestOption) (*Response, error) {
	return c.Request("DELETE", uri, opts...)
}

// Patch issues a PATCH request (HTTP.patch).
func (c *Client) Patch(uri string, opts ...RequestOption) (*Response, error) {
	return c.Request("PATCH", uri, opts...)
}

// Options issues an OPTIONS request (HTTP.options).
func (c *Client) Options(uri string, opts ...RequestOption) (*Response, error) {
	return c.Request("OPTIONS", uri, opts...)
}

// Trace issues a TRACE request (HTTP.trace).
func (c *Client) Trace(uri string, opts ...RequestOption) (*Response, error) {
	return c.Request("TRACE", uri, opts...)
}

// Connect issues a CONNECT request (HTTP.connect).
func (c *Client) Connect(uri string, opts ...RequestOption) (*Response, error) {
	return c.Request("CONNECT", uri, opts...)
}

// Request builds and performs a request for verb/uri, following redirects when
// the client is configured to (HTTP::Client#request). It is the general entry
// point behind the verb methods.
func (c *Client) Request(verb, uri string, opts ...RequestOption) (*Response, error) {
	req := buildRequest(verb, uri, c.options, opts)
	if c.options.Persistent != "" && hostOf(uri) != hostOf(c.options.Persistent) {
		return nil, newError(KindStateError,
			"persistent connection to "+c.options.Persistent+" can't be reused for "+uri)
	}
	resp, err := c.perform(req)
	if err != nil {
		return nil, err
	}
	if c.options.Follow != nil {
		return followRedirects(c.perform, req, resp, c.options.Follow)
	}
	return resp, nil
}

// perform runs a single request through the transport and binds it to the
// resulting response (no redirect following).
func (c *Client) perform(req *Request) (*Response, error) {
	resp, err := c.transport.Perform(req)
	if err != nil {
		return nil, err
	}
	resp.request = req
	return resp, nil
}

// Persistent runs block against a branched client pinned to host, mirroring
// HTTP.persistent(host) { |http| ... }. Requests to a different host inside the
// block fail with a StateError. The block's error (if any) is returned.
func (c *Client) Persistent(host string, block func(*Client) error) error {
	pc := c.branch(func(o *Options) { o.Persistent = host })
	return block(pc)
}

// BasicAuthHeader builds the HTTP Basic Authorization header value for a user and
// password: "Basic " followed by base64("user:password").
func BasicAuthHeader(user, pass string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
}

// acceptType expands http.rb's Accept shorthand: "json"/":json" ⇒
// "application/json"; any other value is used verbatim.
func acceptType(typ string) string {
	switch typ {
	case "json", ":json":
		return "application/json"
	}
	return typ
}

// hostOf returns the host of a URL (scheme://host/…); a value with no host (a
// bare "example.com") is returned unchanged, so a persistent host given with or
// without a scheme compares equal.
func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	return u.Host
}
