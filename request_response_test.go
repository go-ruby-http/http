// Copyright (c) the go-ruby-http/http authors
//
// SPDX-License-Identifier: BSD-3-Clause

package http

import (
	"errors"
	"testing"
)

func TestBuildRequestBodies(t *testing.T) {
	opts := &Options{}
	// raw body wins over everything.
	r := buildRequest("POST", "http://x.test/p", opts,
		[]RequestOption{Body("raw"), Form(NewValues(KV{"a", "1"}))})
	if r.Body != "raw" {
		t.Fatalf("raw body = %q", r.Body)
	}

	// form.
	r = buildRequest("POST", "http://x.test/p", opts, []RequestOption{Form(NewValues(KV{"a", "x y"}))})
	if r.Body != "a=x+y" {
		t.Fatalf("form body = %q", r.Body)
	}
	if ct, _ := r.Headers.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
		t.Fatalf("form ct = %q", ct)
	}

	// json.
	r = buildRequest("POST", "http://x.test/p", opts, []RequestOption{JSON(map[string]any{"a": 1})})
	if r.Body != `{"a":1}` {
		t.Fatalf("json body = %q", r.Body)
	}
	if ct, _ := r.Headers.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("json ct = %q", ct)
	}

	// no body.
	r = buildRequest("POST", "http://x.test/p", opts, nil)
	if r.Body != "" {
		t.Fatalf("empty body = %q", r.Body)
	}
}

func TestBuildRequestGetStripsContentType(t *testing.T) {
	opts := &Options{Headers: NewHeaders(KV{"Content-Type", "application/json"})}
	r := buildRequest("get", "http://x.test/p", opts, nil)
	if r.Verb != "GET" {
		t.Fatalf("verb = %q", r.Verb)
	}
	if r.Headers.Has("Content-Type") {
		t.Fatalf("GET should strip Content-Type")
	}
}

func TestBuildRequestParamsAndCookies(t *testing.T) {
	opts := &Options{Cookies: NewCookieJar()}
	opts.Cookies.Add("sid", "42")
	r := buildRequest("GET", "http://x.test/p?z=1", opts, []RequestOption{Params(NewValues(KV{"q", "a b"}))})
	if r.URL != "http://x.test/p?z=1&q=a+b" {
		t.Fatalf("params url = %q", r.URL)
	}
	if v, _ := r.Headers.Get("Cookie"); v != "sid=42" {
		t.Fatalf("cookie header = %q", v)
	}
	// params on a query-less URL uses '?'.
	r2 := buildRequest("GET", "http://x.test/p", &Options{}, []RequestOption{Params(NewValues(KV{"q", "1"}))})
	if r2.URL != "http://x.test/p?q=1" {
		t.Fatalf("params url2 = %q", r2.URL)
	}
	// empty cookie jar adds no Cookie header.
	r3 := buildRequest("GET", "http://x.test/p", &Options{Cookies: NewCookieJar()}, nil)
	if r3.Headers.Has("Cookie") {
		t.Fatalf("empty jar set Cookie header")
	}
}

func TestResponseAccessors(t *testing.T) {
	h := NewHeaders(KV{"Content-Type", "application/json"})
	h.Add("Set-Cookie", "sid=1")
	r := NewResponse(200, h, `{"ok":true}`, "1.1", "http://x.test/")
	if r.Code() != 200 || r.Reason() != "OK" || !r.Status().Success() {
		t.Fatalf("status accessors wrong")
	}
	if r.Body().String() != `{"ok":true}` {
		t.Fatalf("body = %q", r.Body().String())
	}
	if r.Version() != "1.1" || r.URI() != "http://x.test/" {
		t.Fatalf("version/uri wrong")
	}
	if r.ContentType().MimeType != "application/json" {
		t.Fatalf("content type wrong")
	}
	if v, ok := r.Cookies().Get("sid"); !ok || v != "1" {
		t.Fatalf("cookies = %q,%v", v, ok)
	}
	if r.Headers().Len() == 0 {
		t.Fatalf("headers empty")
	}
	if r.Request() != nil {
		t.Fatalf("bare response should have nil request")
	}
	// nil headers path.
	r2 := NewResponse(204, nil, "", "1.1", "")
	if r2.Headers().Len() != 0 {
		t.Fatalf("nil headers not empty")
	}
}

func TestResponseParse(t *testing.T) {
	h := NewHeaders(KV{"Content-Type", "application/json"})
	r := NewResponse(200, h, `{"a":1}`, "1.1", "")
	v, err := r.Parse()
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	if m, ok := v.(map[string]any); !ok || m["a"] != float64(1) {
		t.Fatalf("parsed = %v", v)
	}
	// override media type.
	r2 := NewResponse(200, NewHeaders(KV{"Content-Type", "text/plain"}), `[1,2]`, "1.1", "")
	if _, err := r2.Parse("json"); err != nil {
		t.Fatalf("override parse err: %v", err)
	}
	// unknown mime.
	_, err = r2.Parse()
	if !errors.Is(err, ErrError) {
		t.Fatalf("unknown mime err = %v", err)
	}
	// malformed JSON.
	r3 := NewResponse(200, h, `{bad`, "1.1", "")
	_, err = r3.Parse()
	if !errors.Is(err, ErrError) {
		t.Fatalf("malformed err = %v", err)
	}
}

func TestIsJSONMime(t *testing.T) {
	for _, m := range []string{"json", ":json", "application/json", "text/json", "application/vnd.api+json"} {
		if !isJSONMime(m) {
			t.Fatalf("%q should be json", m)
		}
	}
	if isJSONMime("text/html") {
		t.Fatalf("text/html not json")
	}
}
