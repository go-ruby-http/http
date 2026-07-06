// Copyright (c) the go-ruby-http/http authors
//
// SPDX-License-Identifier: BSD-3-Clause

package http

import (
	"reflect"
	"testing"
)

func TestNormalizeHeaderName(t *testing.T) {
	cases := map[string]string{
		"content-type":     "Content-Type",
		"CONTENT_TYPE":     "Content-Type",
		"WWW-Authenticate": "Www-Authenticate",
		"ETag":             "Etag",
		"x-custom-header":  "X-Custom-Header",
		"":                 "",
		"-":                "",
	}
	for in, want := range cases {
		if got := NormalizeHeaderName(in); got != want {
			t.Fatalf("NormalizeHeaderName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHeadersSetGetHasDelete(t *testing.T) {
	h := NewHeaders(KV{"Accept", "a"})
	h.Set("content-type", "text/plain")
	h.Set("CONTENT-TYPE", "application/json") // replace, single value
	if h.Len() != 2 {
		t.Fatalf("len = %d, want 2", h.Len())
	}
	if v, ok := h.Get("Content-Type"); !ok || v != "application/json" {
		t.Fatalf("get = %q,%v", v, ok)
	}
	if _, ok := h.Get("Missing"); ok {
		t.Fatalf("missing get returned ok")
	}
	if !h.Has("content-TYPE") || h.Has("Nope") {
		t.Fatalf("Has broken")
	}
	if p := h.Pairs(); p[1].Key != "Content-Type" {
		t.Fatalf("normalized key = %q", p[1].Key)
	}
	h.Delete("nope-not-there") // no-op branch
	h.Delete("Accept")
	if h.Has("Accept") {
		t.Fatalf("delete failed")
	}
}

func TestHeadersAddGetAll(t *testing.T) {
	h := NewHeaders()
	h.Add("Set-Cookie", "a=1")
	h.Add("set-cookie", "b=2")
	all := h.GetAll("Set-Cookie")
	if !reflect.DeepEqual(all, []string{"a=1", "b=2"}) {
		t.Fatalf("GetAll = %v", all)
	}
	if v, ok := h.Get("Set-Cookie"); !ok || v != "a=1" {
		t.Fatalf("Get first = %q,%v", v, ok)
	}
	if got := h.GetAll("None"); got != nil {
		t.Fatalf("GetAll missing = %v", got)
	}
}

func TestHeadersSetDefault(t *testing.T) {
	h := NewHeaders()
	h.SetDefault("Accept", "a")
	h.SetDefault("accept", "b") // present: no clobber
	if v, _ := h.Get("Accept"); v != "a" {
		t.Fatalf("SetDefault clobbered: %q", v)
	}
}

func TestHeadersCloneMerge(t *testing.T) {
	h := NewHeaders(KV{"A", "1"})
	c := h.Clone()
	c.Set("A", "2")
	if v, _ := h.Get("A"); v != "1" {
		t.Fatalf("clone shared state: %q", v)
	}
	m := h.Merge(NewHeaders(KV{"A", "9"}, KV{"B", "3"}))
	if v, _ := m.Get("A"); v != "9" {
		t.Fatalf("merge overwrite: %q", v)
	}
	if v, _ := m.Get("B"); v != "3" {
		t.Fatalf("merge add: %q", v)
	}
	if m2 := h.Merge(nil); m2.Len() != 1 {
		t.Fatalf("merge nil: len %d", m2.Len())
	}
}
