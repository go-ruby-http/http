// Copyright (c) the go-ruby-http/http authors
//
// SPDX-License-Identifier: BSD-3-Clause

package http

import (
	"errors"
	"io"
	nethttp "net/http"
	"strings"
	"testing"
)

// fakeHTTP is a stub httpClient: it records the request and returns a canned
// response/error, so the NetTransport mapping is exercised without a socket.
type fakeHTTP struct {
	resp   *nethttp.Response
	err    error
	gotReq *nethttp.Request
}

func (f *fakeHTTP) Do(r *nethttp.Request) (*nethttp.Response, error) {
	f.gotReq = r
	return f.resp, f.err
}

// timeoutErr implements the net timeout surface.
type timeoutErr struct{}

func (timeoutErr) Error() string { return "i/o timeout" }
func (timeoutErr) Timeout() bool { return true }

// errReadCloser fails on Read to exercise the body-read error path.
type errReadCloser struct{}

func (errReadCloser) Read([]byte) (int, error) { return 0, errors.New("read boom") }
func (errReadCloser) Close() error             { return nil }

func mkResp(code int, body string) *nethttp.Response {
	return &nethttp.Response{
		StatusCode: code,
		Proto:      "HTTP/1.1",
		Header:     nethttp.Header{"Content-Type": {"application/json"}, "X-Multi": {"a", "b"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestNetTransportSuccess(t *testing.T) {
	fc := &fakeHTTP{resp: mkResp(201, `{"ok":true}`)}
	tr := &NetTransport{Client: fc}
	req := &Request{Verb: "POST", URL: "http://x.test/p", Headers: NewHeaders(KV{"X-Trace", "1"}), Body: "hi"}
	resp, err := tr.Perform(req)
	if err != nil {
		t.Fatalf("perform err: %v", err)
	}
	if resp.Code() != 201 || resp.Version() != "1.1" {
		t.Fatalf("resp code/version wrong: %d %q", resp.Code(), resp.Version())
	}
	if v, _ := resp.Headers().Get("X-Multi"); v != "a" {
		t.Fatalf("multi header lost: %q", v)
	}
	if fc.gotReq.Header.Get("X-Trace") != "1" {
		t.Fatalf("request header not forwarded")
	}
	if b, _ := io.ReadAll(fc.gotReq.Body); string(b) != "hi" {
		t.Fatalf("request body not forwarded: %q", b)
	}
}

func TestNetTransportEmptyBody(t *testing.T) {
	fc := &fakeHTTP{resp: mkResp(200, "")}
	tr := &NetTransport{Client: fc}
	req := &Request{Verb: "GET", URL: "http://x.test/", Headers: NewHeaders()}
	if _, err := tr.Perform(req); err != nil {
		t.Fatalf("perform err: %v", err)
	}
	if fc.gotReq.Body != nil {
		t.Fatalf("empty body should yield nil request body")
	}
}

func TestNetTransportNewRequestError(t *testing.T) {
	tr := &NetTransport{Client: &fakeHTTP{}}
	req := &Request{Verb: "BAD METHOD", URL: "http://x.test/", Headers: NewHeaders()}
	_, err := tr.Perform(req)
	if !errors.Is(err, ErrConnectionError) {
		t.Fatalf("expected ConnectionError, got %v", err)
	}
}

func TestNetTransportTimeout(t *testing.T) {
	tr := &NetTransport{Client: &fakeHTTP{err: timeoutErr{}}}
	req := &Request{Verb: "GET", URL: "http://x.test/", Headers: NewHeaders()}
	_, err := tr.Perform(req)
	if !errors.Is(err, ErrTimeoutError) {
		t.Fatalf("expected TimeoutError, got %v", err)
	}
}

func TestNetTransportConnFailed(t *testing.T) {
	tr := &NetTransport{Client: &fakeHTTP{err: errors.New("refused")}}
	req := &Request{Verb: "GET", URL: "http://x.test/", Headers: NewHeaders()}
	_, err := tr.Perform(req)
	if !errors.Is(err, ErrConnectionError) {
		t.Fatalf("expected ConnectionError, got %v", err)
	}
}

func TestNetTransportBodyReadError(t *testing.T) {
	resp := &nethttp.Response{StatusCode: 200, Proto: "HTTP/2.0", Header: nethttp.Header{}, Body: errReadCloser{}}
	tr := &NetTransport{Client: &fakeHTTP{resp: resp}}
	req := &Request{Verb: "GET", URL: "http://x.test/", Headers: NewHeaders()}
	_, err := tr.Perform(req)
	if !errors.Is(err, ErrConnectionError) {
		t.Fatalf("expected ConnectionError on read failure, got %v", err)
	}
}

func TestTransportFuncAndDefault(t *testing.T) {
	called := false
	var tr Transport = TransportFunc(func(req *Request) (*Response, error) {
		called = true
		return NewResponse(200, nil, "", "1.1", req.URL), nil
	})
	if _, err := tr.Perform(&Request{URL: "u"}); err != nil || !called {
		t.Fatalf("TransportFunc not invoked")
	}
	if DefaultTransport().Client == nil {
		t.Fatalf("DefaultTransport has no client")
	}
}

func TestHTTPVersion(t *testing.T) {
	if httpVersion("HTTP/1.1") != "1.1" {
		t.Fatalf("version strip failed")
	}
	if httpVersion("weird") != "weird" {
		t.Fatalf("version passthrough failed")
	}
}
