// Copyright (c) the go-ruby-http/http authors
//
// SPDX-License-Identifier: BSD-3-Clause

package http

import (
	"io"
	nethttp "net/http"
	"strings"
)

// Transport is the host seam that performs the HTTP round-trip, mirroring the
// role http.rb's HTTP::Client delegates to its socket/connection. Given a
// prepared [Request], a Transport returns the finished [Response] or a transport
// [Error] (ConnectionError / TimeoutError).
//
// The default production Transport is [DefaultTransport] (backed by net/http).
// The core opens no socket itself: every request runs through whatever Transport
// the client is configured with, so tests inject a [TransportFunc] stub and
// never touch the network.
type Transport interface {
	Perform(req *Request) (*Response, error)
}

// TransportFunc adapts a function to the [Transport] interface — the convenient
// way to inject a stub in tests or a custom transport in a host.
type TransportFunc func(req *Request) (*Response, error)

// Perform invokes f(req).
func (f TransportFunc) Perform(req *Request) (*Response, error) { return f(req) }

// httpClient is the minimal net/http surface [NetTransport] depends on,
// indirected so tests can drive the transport's request-building and
// response/error mapping without opening a socket.
type httpClient interface {
	Do(req *nethttp.Request) (*nethttp.Response, error)
}

// NetTransport is the default [Transport]: it turns a [Request] into a net/http
// request, executes it with its Client, and maps the response (or a transport
// failure) back to a [Response] or an [Error]. A timeout maps to a TimeoutError,
// any other transport failure to a ConnectionError.
type NetTransport struct {
	// Client performs the request; defaults to http.DefaultClient.
	Client httpClient
}

// DefaultTransport returns the default net/http-backed [Transport].
func DefaultTransport() *NetTransport { return &NetTransport{Client: nethttp.DefaultClient} }

// Perform executes req with net/http and maps the outcome.
func (t *NetTransport) Perform(req *Request) (*Response, error) {
	var body io.Reader
	if req.Body != "" {
		body = strings.NewReader(req.Body)
	}
	hr, err := nethttp.NewRequest(req.Verb, req.URL, body)
	if err != nil {
		return nil, newTransportError(KindConnectionError, err)
	}
	for _, p := range req.Headers.Pairs() {
		hr.Header.Add(p.Key, p.Val)
	}

	resp, err := t.Client.Do(hr)
	if err != nil {
		return nil, newTransportError(classifyTransport(err), err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, newTransportError(KindConnectionError, err)
	}
	return NewResponse(resp.StatusCode, headersFromHTTP(resp.Header), string(raw),
		httpVersion(resp.Proto), req.URL), nil
}

// timeoutError is the net-error surface used to detect a timeout.
type timeoutError interface{ Timeout() bool }

// classifyTransport maps a net/http client error to a transport error kind: a
// timeout becomes TimeoutError, anything else ConnectionError.
func classifyTransport(err error) ErrorKind {
	if te, ok := err.(timeoutError); ok && te.Timeout() {
		return KindTimeoutError
	}
	return KindConnectionError
}

// headersFromHTTP converts an http.Header into a [Headers], preserving every
// value (multi-valued headers keep their entries).
func headersFromHTTP(h nethttp.Header) *Headers {
	out := NewHeaders()
	for k, vs := range h {
		for _, v := range vs {
			out.Add(k, v)
		}
	}
	return out
}

// httpVersion strips the "HTTP/" prefix from an http.Response.Proto ("HTTP/1.1"
// → "1.1"); a value without the prefix is returned unchanged.
func httpVersion(proto string) string {
	return strings.TrimPrefix(proto, "HTTP/")
}
