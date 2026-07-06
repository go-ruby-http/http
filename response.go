// Copyright (c) the go-ruby-http/http authors
//
// SPDX-License-Identifier: BSD-3-Clause

package http

import (
	"encoding/json"
	"strings"
)

// Response is the result of a request, mirroring HTTP::Response: the status
// (with its predicates), headers, body, parsed content type, cookies, and the
// version/URI. Build one with [NewResponse] (a transport does this); read it via
// the accessors.
type Response struct {
	status  Status
	headers *Headers
	body    *ResponseBody
	version string
	uri     string
	request *Request
	cookies *CookieJar
}

// NewResponse builds a [Response] from the parts a transport produces. The
// cookies are derived from the Set-Cookie response headers. A nil headers is
// treated as empty.
func NewResponse(status int, headers *Headers, body, version, uri string) *Response {
	if headers == nil {
		headers = NewHeaders()
	}
	return &Response{
		status:  Status(status),
		headers: headers,
		body:    newResponseBody(body),
		version: version,
		uri:     uri,
		cookies: cookieJarFromHeaders(headers),
	}
}

// Status returns the [Status] (HTTP::Response#status).
func (r *Response) Status() Status { return r.status }

// Code returns the numeric status code (HTTP::Response#code).
func (r *Response) Code() int { return r.status.Code() }

// Reason returns the status reason phrase (HTTP::Response#reason).
func (r *Response) Reason() string { return r.status.Reason() }

// Headers returns the response headers (HTTP::Response#headers).
func (r *Response) Headers() *Headers { return r.headers }

// Body returns the response body (HTTP::Response#body).
func (r *Response) Body() *ResponseBody { return r.body }

// Version returns the HTTP version string (HTTP::Response#version), e.g. "1.1".
func (r *Response) Version() string { return r.version }

// URI returns the effective request URI as a string (HTTP::Response#uri).
func (r *Response) URI() string { return r.uri }

// Request returns the [Request] that produced this response, when the client set
// it (HTTP::Response#request); nil for a bare [NewResponse].
func (r *Response) Request() *Request { return r.request }

// Cookies returns the cookies parsed from the response's Set-Cookie headers
// (HTTP::Response#cookies).
func (r *Response) Cookies() *CookieJar { return r.cookies }

// ContentType returns the parsed Content-Type header (HTTP::Response#content_type).
// A missing header yields the zero [ContentType].
func (r *Response) ContentType() ContentType {
	v, _ := r.headers.Get("Content-Type")
	return ParseContentType(v)
}

// Parse decodes the body according to its media type, mirroring
// HTTP::Response#parse. With no argument it uses the response's Content-Type; an
// optional override forces a media type (":json"/"json"/"application/json" all
// select JSON). Only JSON is decoded by the deterministic core; any other media
// type yields an [Error] of kind [KindError] ("unknown MIME type"), and a
// malformed JSON body yields a [KindError] wrapping the decode failure.
func (r *Response) Parse(as ...string) (any, error) {
	mime := r.ContentType().MimeType
	if len(as) > 0 {
		mime = strings.ToLower(strings.TrimSpace(as[0]))
	}
	if !isJSONMime(mime) {
		return nil, newError(KindError, "unknown MIME type: "+mime)
	}
	var v any
	if err := json.Unmarshal([]byte(r.body.s), &v); err != nil {
		return nil, &Error{Kind: KindError, Message: err.Error(), Cause: err}
	}
	return v, nil
}

// isJSONMime reports whether a media type selects the JSON adapter: the bare
// aliases "json"/":json", application/json, text/json, or any "+json"
// structured-suffix type.
func isJSONMime(mime string) bool {
	switch mime {
	case "json", ":json", "application/json", "text/json":
		return true
	}
	return strings.HasSuffix(mime, "+json")
}
