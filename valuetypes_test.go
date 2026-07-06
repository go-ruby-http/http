// Copyright (c) the go-ruby-http/http authors
//
// SPDX-License-Identifier: BSD-3-Clause

package http

import "testing"

func TestContentType(t *testing.T) {
	cases := []struct {
		in, mime, charset string
	}{
		{"application/json; charset=utf-8", "application/json", "utf-8"},
		{"text/html", "text/html", ""},
		{"; charset=x", "", "x"},
		{`TEXT/HTML; Charset="UTF-8"`, "text/html", "UTF-8"},
		{"  application/JSON  ", "application/json", ""},
		{"", "", ""},
		{"text/html; boundary=z", "text/html", ""},
	}
	for _, c := range cases {
		ct := ParseContentType(c.in)
		if ct.MimeType != c.mime || ct.Charset != c.charset {
			t.Fatalf("ParseContentType(%q) = %q/%q, want %q/%q", c.in, ct.MimeType, ct.Charset, c.mime, c.charset)
		}
	}
}

func TestResponseBody(t *testing.T) {
	b := newResponseBody("hello")
	if b.String() != "hello" || b.To_s() != "hello" {
		t.Fatalf("body accessors wrong")
	}
}

func TestValuesEncode(t *testing.T) {
	v := NewValues(KV{"a", "1"}).Add("b", "x y").Add("c", "1").Add("c", "2")
	if got := v.Encode(); got != "a=1&b=x+y&c[]=1&c[]=2" {
		t.Fatalf("Encode = %q", got)
	}
	if v.Len() != 3 {
		t.Fatalf("Len = %d", v.Len())
	}
	if val, ok := v.Get("b"); !ok || val != "x y" {
		t.Fatalf("Get = %q,%v", val, ok)
	}
	if _, ok := v.Get("z"); ok {
		t.Fatalf("Get missing ok")
	}
}

func TestValuesSetAndNil(t *testing.T) {
	var v Values // zero value exercises ensure()
	v.Set("k", "1")
	v.Set("k", "2") // existing key replace
	v.AddNil("bare")
	v.AddNil("bare") // existing nil key no-op
	if got := v.Encode(); got != "k=2&bare" {
		t.Fatalf("Encode = %q", got)
	}
	// Set clears a prior nil marker.
	var w Values
	w.AddNil("x")
	w.Set("x", "v")
	if got := w.Encode(); got != "x=v" {
		t.Fatalf("Encode after Set-over-nil = %q", got)
	}
	// Get on a key with no stored value.
	var z Values
	z.AddNil("n")
	if val, ok := z.Get("n"); ok || val != "" {
		t.Fatalf("Get bare = %q,%v", val, ok)
	}
}

func TestEscapeFormComponent(t *testing.T) {
	if got := EscapeFormComponent("~-_.!* /a"); got != "%7E-_.%21*+%2Fa" {
		t.Fatalf("escape = %q", got)
	}
	if got := EscapeFormComponent("Z9"); got != "Z9" {
		t.Fatalf("escape alnum = %q", got)
	}
}

func TestCookieJar(t *testing.T) {
	j := NewCookieJar()
	j.ParseSetCookie("foo=bar; Path=/; HttpOnly")
	j.ParseSetCookie("baz=qux")
	j.ParseSetCookie("noeq")     // skipped (no '=')
	j.ParseSetCookie("=novalue") // skipped (empty name)
	j.Add("foo", "updated")      // overwrite existing
	if v, ok := j.Get("foo"); !ok || v != "updated" {
		t.Fatalf("Get foo = %q,%v", v, ok)
	}
	if _, ok := j.Get("nope"); ok {
		t.Fatalf("Get missing ok")
	}
	if j.Len() != 2 {
		t.Fatalf("Len = %d, want 2", j.Len())
	}
	if got := j.HeaderValue(); got != "foo=updated; baz=qux" {
		t.Fatalf("HeaderValue = %q", got)
	}
	if len(j.Cookies()) != 2 {
		t.Fatalf("Cookies len")
	}
	c := j.Clone()
	c.Add("baz", "changed")
	if v, _ := j.Get("baz"); v != "qux" {
		t.Fatalf("clone shared state")
	}
	if NewCookieJar().HeaderValue() != "" {
		t.Fatalf("empty jar header")
	}
}

func TestCookieJarLazyIndex(t *testing.T) {
	var j CookieJar // nil index exercises the lazy-init branch in Add
	j.Add("a", "1")
	if v, _ := j.Get("a"); v != "1" {
		t.Fatalf("lazy add failed")
	}
}
