// Copyright (c) the go-ruby-http/http authors
//
// SPDX-License-Identifier: BSD-3-Clause

package http

// ResponseBody is the body of a [Response], mirroring http.rb's
// HTTP::Response::Body. The deterministic core carries the fully-read body as a
// string; a streaming host transport may fill it lazily, but callers observe it
// through [ResponseBody.String]/[ResponseBody.To_s].
type ResponseBody struct {
	s string
}

// newResponseBody wraps s as a body.
func newResponseBody(s string) *ResponseBody { return &ResponseBody{s: s} }

// String returns the body as a string (HTTP::Response::Body#to_s). It also
// satisfies fmt.Stringer.
func (b *ResponseBody) String() string { return b.s }

// To_s is the http.rb-named alias for [ResponseBody.String] (Body#to_s), for
// hosts that surface the Ruby method name.
func (b *ResponseBody) To_s() string { return b.s } //nolint:revive
