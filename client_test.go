// Copyright (c) the go-ruby-http/http authors
//
// SPDX-License-Identifier: BSD-3-Clause

package http

import (
	"errors"
	"testing"
)

// recorder is a stub transport recording every request and replaying a script of
// (response, error) steps; the last step repeats once exhausted.
type recorder struct {
	steps []step
	i     int
	reqs  []*Request
}

type step struct {
	resp *Response
	err  error
}

func (r *recorder) Perform(req *Request) (*Response, error) {
	r.reqs = append(r.reqs, req)
	s := r.steps[r.i]
	if r.i < len(r.steps)-1 {
		r.i++
	}
	if s.resp != nil {
		s.resp.request = req // mimic a fresh response per hop
	}
	return s.resp, s.err
}

func mkReq(verb, url string) *Request {
	return &Request{Verb: verb, URL: url, Headers: NewHeaders()}
}

func okResp(code int, loc string) *Response {
	h := NewHeaders(KV{"Content-Type", "text/plain"})
	if loc != "" {
		h.Set("Location", loc)
	}
	return NewResponse(code, h, "body", "1.1", "http://x.test/")
}

func TestChainableBuildsHeaders(t *testing.T) {
	rec := &recorder{steps: []step{{resp: okResp(200, "")}}}
	c := NewClient().WithTransport(rec).
		Headers(KV{"X-A", "1"}).
		Header("X-B", "2").
		Auth("Bearer tok").
		Accept("json").
		Cookies(KV{"sid", "9"}).
		Timeout(30).
		TimeoutOps(1, 2, 3).
		Encoding("UTF-8").
		Nodelay().
		Use("logging").
		Via("proxy.test", 8080, "pu", "pp")
	if _, err := c.Get("http://x.test/p"); err != nil {
		t.Fatalf("get err: %v", err)
	}
	req := rec.reqs[0]
	want := map[string]string{
		"X-A": "1", "X-B": "2", "Authorization": "Bearer tok",
		"Accept": "application/json", "Cookie": "sid=9",
	}
	for k, v := range want {
		if got, _ := req.Headers.Get(k); got != v {
			t.Fatalf("header %s = %q, want %q", k, got, v)
		}
	}
	if req.Proxy == nil || req.Proxy.Host != "proxy.test" || req.Proxy.User != "pu" || req.Proxy.Pass != "pp" {
		t.Fatalf("proxy not set: %+v", req.Proxy)
	}
	// Options snapshot reflects the branch mutations.
	o := c.DefaultOptions()
	if o.Timeout == nil || o.Timeout.Mode != "per_operation" || !o.Nodelay || o.Encoding != "UTF-8" ||
		len(o.Features) != 1 {
		t.Fatalf("options not branched: %+v", o)
	}
}

func TestChainableAcceptAndBasicAuthAndThrough(t *testing.T) {
	rec := &recorder{steps: []step{{resp: okResp(200, "")}}}
	c := NewClient().WithTransport(rec).BasicAuth("u", "p").Accept("text/html").Through("p.test", 3128)
	if _, err := c.Get("http://x.test/"); err != nil {
		t.Fatalf("err %v", err)
	}
	req := rec.reqs[0]
	if v, _ := req.Headers.Get("Authorization"); v != "Basic dTpw" {
		t.Fatalf("basic auth = %q", v)
	}
	if v, _ := req.Headers.Get("Accept"); v != "text/html" {
		t.Fatalf("accept verbatim = %q", v)
	}
	if req.Proxy.Host != "p.test" || req.Proxy.Port != 3128 {
		t.Fatalf("through proxy = %+v", req.Proxy)
	}
	// :json alias.
	if acceptType(":json") != "application/json" {
		t.Fatalf(":json alias")
	}
}

func TestAllVerbs(t *testing.T) {
	rec := &recorder{steps: []step{{resp: okResp(200, "")}}}
	c := NewClient().WithTransport(rec)
	type call func(string, ...RequestOption) (*Response, error)
	verbs := map[string]call{
		"GET": c.Get, "HEAD": c.Head, "POST": c.Post, "PUT": c.Put,
		"DELETE": c.Delete, "PATCH": c.Patch, "OPTIONS": c.Options,
		"TRACE": c.Trace, "CONNECT": c.Connect,
	}
	for want, fn := range verbs {
		rec.reqs = nil
		if _, err := fn("http://x.test/"); err != nil {
			t.Fatalf("%s err: %v", want, err)
		}
		if rec.reqs[0].Verb != want {
			t.Fatalf("verb = %q, want %q", rec.reqs[0].Verb, want)
		}
	}
}

func TestRequestTransportError(t *testing.T) {
	rec := &recorder{steps: []step{{err: newTransportError(KindConnectionError, errors.New("x"))}}}
	_, err := NewClient().WithTransport(rec).Get("http://x.test/")
	if !errors.Is(err, ErrConnectionError) {
		t.Fatalf("expected ConnectionError, got %v", err)
	}
}

func TestFollowRedirectSuccess(t *testing.T) {
	rec := &recorder{steps: []step{
		{resp: okResp(302, "http://x.test/next")},
		{resp: okResp(200, "")},
	}}
	resp, err := NewClient().WithTransport(rec).Follow().Get("http://x.test/start")
	if err != nil {
		t.Fatalf("follow err: %v", err)
	}
	if resp.Code() != 200 {
		t.Fatalf("final code = %d", resp.Code())
	}
	if len(rec.reqs) != 2 || rec.reqs[1].URL != "http://x.test/next" {
		t.Fatalf("redirect target wrong: %+v", rec.reqs)
	}
}

func TestFollowNoRedirect(t *testing.T) {
	rec := &recorder{steps: []step{{resp: okResp(200, "")}}}
	resp, err := NewClient().WithTransport(rec).Follow(FollowOptions{MaxHops: 3, Strict: true}).Get("http://x.test/")
	if err != nil || resp.Code() != 200 {
		t.Fatalf("no-redirect follow failed: %v %d", err, resp.Code())
	}
}

func TestFollowTooManyHops(t *testing.T) {
	// Each hop points to a distinct new URL so no endless-loop triggers first.
	tr := TransportFunc(func(req *Request) (*Response, error) {
		return okResp(302, req.URL+"/x"), nil
	})
	_, err := NewClient().WithTransport(tr).Follow(FollowOptions{MaxHops: 1, Strict: true}).Get("http://x.test/a")
	if !errors.Is(err, ErrResponseError) {
		t.Fatalf("expected ResponseError (too many), got %v", err)
	}
}

func TestFollowEndlessLoop(t *testing.T) {
	// Always redirect to the same absolute URL → identical visit key → loop.
	tr := TransportFunc(func(req *Request) (*Response, error) {
		return okResp(302, "http://x.test/loop"), nil
	})
	_, err := NewClient().WithTransport(tr).Follow().Get("http://x.test/loop")
	if !errors.Is(err, ErrResponseError) {
		t.Fatalf("expected ResponseError (endless), got %v", err)
	}
}

func TestFollowPerformErrorMidChain(t *testing.T) {
	rec := &recorder{steps: []step{
		{resp: okResp(302, "http://x.test/next")},
		{err: newTransportError(KindTimeoutError, errors.New("t"))},
	}}
	_, err := NewClient().WithTransport(rec).Follow().Get("http://x.test/start")
	if !errors.Is(err, ErrTimeoutError) {
		t.Fatalf("expected TimeoutError mid-chain, got %v", err)
	}
}

func TestFollowDefaultMaxHopsAndNoLocation(t *testing.T) {
	// MaxHops:0 exercises the default-hops branch in followRedirects.
	rec := &recorder{steps: []step{
		{resp: okResp(302, "http://x.test/next")},
		{resp: okResp(200, "")},
	}}
	resp, err := followRedirects(rec.Perform, mkReq("GET", "http://x.test/a"), okResp(302, "http://x.test/a2"),
		&FollowOptions{MaxHops: 0, Strict: true})
	if err != nil || resp.Code() != 200 {
		t.Fatalf("default-hops follow failed: %v", err)
	}

	// A redirect during a follow whose response lacks Location surfaces the
	// StateError from redirectTo.
	rec2 := &recorder{steps: []step{{resp: NewResponse(302, NewHeaders(), "", "1.1", "http://x.test/")}}}
	_, err = followRedirects(rec2.Perform, mkReq("GET", "http://x.test/a"),
		NewResponse(302, NewHeaders(), "", "1.1", "http://x.test/"), &FollowOptions{Strict: true})
	if !errors.Is(err, ErrStateError) {
		t.Fatalf("expected StateError from missing Location mid-follow, got %v", err)
	}
}

func TestRedirectToNoLocation(t *testing.T) {
	resp := NewResponse(302, NewHeaders(), "", "1.1", "http://x.test/")
	req := &Request{Verb: "GET", URL: "http://x.test/", Headers: NewHeaders()}
	if _, err := redirectTo(req, resp, true); !errors.Is(err, ErrStateError) {
		t.Fatalf("expected StateError, got %v", err)
	}
}

func TestRedirectToUnsafeStrict(t *testing.T) {
	resp := okResp(302, "http://x.test/next")
	req := &Request{Verb: "POST", URL: "http://x.test/", Headers: NewHeaders(), Body: "b"}
	if _, err := redirectTo(req, resp, true); !errors.Is(err, ErrStateError) {
		t.Fatalf("strict POST 302 should StateError, got %v", err)
	}
}

func TestRedirectToUnsafeNonStrictBecomesGet(t *testing.T) {
	resp := okResp(302, "http://x.test/next")
	req := &Request{Verb: "POST", URL: "http://x.test/", Headers: NewHeaders(KV{"Content-Type", "application/json"}), Body: "b"}
	next, err := redirectTo(req, resp, false)
	if err != nil {
		t.Fatalf("err %v", err)
	}
	if next.Verb != "GET" || next.Body != "" || next.Headers.Has("Content-Type") {
		t.Fatalf("non-strict downgrade wrong: %+v", next)
	}
}

func TestRedirectTo303BecomesGet(t *testing.T) {
	resp := okResp(303, "http://x.test/next")
	req := &Request{Verb: "PUT", URL: "http://x.test/", Headers: NewHeaders(), Body: "b"}
	next, err := redirectTo(req, resp, false)
	if err != nil || next.Verb != "GET" {
		t.Fatalf("303 should become GET: %v %+v", err, next)
	}
}

func TestRedirectToCrossOriginStripsCreds(t *testing.T) {
	resp := okResp(307, "http://other.test/p")
	h := NewHeaders(KV{"Authorization", "secret"}, KV{"Cookie", "sid=1"})
	req := &Request{Verb: "GET", URL: "http://x.test/", Headers: h}
	next, err := redirectTo(req, resp, true)
	if err != nil {
		t.Fatalf("err %v", err)
	}
	if next.Headers.Has("Authorization") || next.Headers.Has("Cookie") {
		t.Fatalf("cross-origin creds not stripped: %+v", next.Headers)
	}
	// Same-origin keeps them.
	resp2 := okResp(307, "http://x.test/p2")
	next2, _ := redirectTo(req, resp2, true)
	if !next2.Headers.Has("Authorization") {
		t.Fatalf("same-origin should keep Authorization")
	}
}

func TestResolveAndOrigin(t *testing.T) {
	if got := resolveURL("http://x.test/a/b", "../c"); got != "http://x.test/c" {
		t.Fatalf("resolve = %q", got)
	}
	if got := resolveURL("http://\x7f", "y"); got != "y" {
		t.Fatalf("bad base fallback = %q", got)
	}
	if got := resolveURL("http://x.test/", "ht\x7ftp://y"); got != "ht\x7ftp://y" {
		t.Fatalf("bad loc fallback = %q", got)
	}
	if sameOrigin("http://\x7f", "http://x.test") {
		t.Fatalf("bad a origin")
	}
	if sameOrigin("http://x.test", "http://\x7f") {
		t.Fatalf("bad b origin")
	}
	if !sameOrigin("http://x.test/a", "http://x.test/b") {
		t.Fatalf("same origin should match")
	}
}

func TestPersistent(t *testing.T) {
	rec := &recorder{steps: []step{{resp: okResp(200, "")}}}
	base := NewClient().WithTransport(rec)
	// Same host succeeds, different host raises StateError.
	err := base.Persistent("http://x.test", func(c *Client) error {
		if _, e := c.Get("http://x.test/a"); e != nil {
			return e
		}
		_, e := c.Get("http://other.test/b")
		return e
	})
	if !errors.Is(err, ErrStateError) {
		t.Fatalf("expected StateError for cross-host persistent, got %v", err)
	}
	// Package-level Persistent + scheme-less host equivalence.
	DefaultClientTransport = rec
	defer func() { DefaultClientTransport = DefaultTransport() }()
	err = Persistent("x.test", func(c *Client) error {
		_, e := c.Get("http://x.test/ok")
		return e
	})
	if err != nil {
		t.Fatalf("scheme-less persistent host mismatch: %v", err)
	}
}

func TestHostOf(t *testing.T) {
	if hostOf("http://x.test/p") != "x.test" {
		t.Fatalf("hostOf scheme")
	}
	if hostOf("x.test") != "x.test" {
		t.Fatalf("hostOf bare")
	}
	if hostOf("http://\x7f") != "http://\x7f" {
		t.Fatalf("hostOf bad url should return raw")
	}
}

func TestOptionsCloneAllFields(t *testing.T) {
	// A fully-configured client exercises every non-nil branch of Options.clone
	// when branched again.
	rec := &recorder{steps: []step{{resp: okResp(200, "")}}}
	c := NewClient().WithTransport(rec).
		Headers(KV{"A", "1"}).Cookies(KV{"c", "1"}).Via("p", 1).
		Follow().Timeout(5).Use("f")
	branched := c.Header("B", "2") // triggers clone of all set fields
	if _, ok := branched.DefaultOptions().Headers.Get("A"); !ok {
		t.Fatalf("clone lost headers")
	}
	// Original unaffected by branch.
	if branched.DefaultOptions().Cookies == c.DefaultOptions().Cookies {
		t.Fatalf("cookies not deep-cloned")
	}
}

func TestPackageLevelChainEntrypoints(t *testing.T) {
	rec := &recorder{steps: []step{{resp: okResp(200, "")}}}
	DefaultClientTransport = rec
	defer func() { DefaultClientTransport = DefaultTransport() }()

	// Chain entrypoints return a client; drive one request through each family.
	clients := []*Client{
		Header("X", "1"), Auth("t"), BasicAuth("u", "p"), Accept("json"),
		Timeout(1), Follow(), Via("p", 1), Through("p", 1),
		Cookies(KV{"c", "1"}), Encoding("UTF-8"), Nodelay(), Use("f"),
	}
	for i, c := range clients {
		if _, err := c.Get("http://x.test/"); err != nil {
			t.Fatalf("client %d get err: %v", i, err)
		}
	}

	// Verb shortcuts.
	verbCalls := []func(string, ...RequestOption) (*Response, error){
		Get, Head, Post, Put, Delete, Patch,
	}
	for i, fn := range verbCalls {
		if _, err := fn("http://x.test/"); err != nil {
			t.Fatalf("verb shortcut %d err: %v", i, err)
		}
	}
}
