// Copyright (c) the go-ruby-http/http authors
//
// SPDX-License-Identifier: BSD-3-Clause

package http

import "strings"

// Cookie is a single name/value cookie, the deterministic subset of
// HTTP::Cookie the client needs to round-trip a session (attributes such as
// Path/Domain/Expires are host concerns and are not modeled).
type Cookie struct {
	Name  string
	Value string
}

// CookieJar is http.rb's HTTP::CookieJar reduced to its deterministic core: an
// ordered set of name/value cookies. It parses Set-Cookie response headers into
// cookies and renders a Cookie request header. Later additions of an existing
// name overwrite the value in place.
type CookieJar struct {
	cookies []Cookie
	index   map[string]int
}

// NewCookieJar returns an empty jar.
func NewCookieJar() *CookieJar { return &CookieJar{index: map[string]int{}} }

// Add inserts or overwrites the cookie named name (HTTP::CookieJar#add).
func (j *CookieJar) Add(name, value string) {
	if j.index == nil {
		j.index = map[string]int{}
	}
	if i, ok := j.index[name]; ok {
		j.cookies[i].Value = value
		return
	}
	j.index[name] = len(j.cookies)
	j.cookies = append(j.cookies, Cookie{Name: name, Value: value})
}

// Get returns the value of the named cookie and whether present.
func (j *CookieJar) Get(name string) (string, bool) {
	if i, ok := j.index[name]; ok {
		return j.cookies[i].Value, true
	}
	return "", false
}

// Len reports the number of cookies.
func (j *CookieJar) Len() int { return len(j.cookies) }

// Cookies returns the jar's cookies in insertion order. The slice must not be
// mutated.
func (j *CookieJar) Cookies() []Cookie { return j.cookies }

// ParseSetCookie parses a single Set-Cookie header value and stores the
// name/value pair (the first "name=value" segment; attributes after the first
// ';' are ignored). A value with no '=' or an empty name is skipped.
func (j *CookieJar) ParseSetCookie(header string) {
	first := header
	if i := strings.IndexByte(first, ';'); i >= 0 {
		first = first[:i]
	}
	name, value, ok := strings.Cut(first, "=")
	name = strings.TrimSpace(name)
	if !ok || name == "" {
		return
	}
	j.Add(name, strings.TrimSpace(value))
}

// HeaderValue renders the jar as a Cookie request-header value:
// "name=value; name2=value2" in insertion order (empty ⇒ "").
func (j *CookieJar) HeaderValue() string {
	var b strings.Builder
	for i, c := range j.cookies {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(c.Name)
		b.WriteByte('=')
		b.WriteString(c.Value)
	}
	return b.String()
}

// Clone returns a deep copy of the jar.
func (j *CookieJar) Clone() *CookieJar {
	c := NewCookieJar()
	for _, k := range j.cookies {
		c.Add(k.Name, k.Value)
	}
	return c
}

// cookieJarFromHeaders builds a jar from every Set-Cookie value in h.
func cookieJarFromHeaders(h *Headers) *CookieJar {
	j := NewCookieJar()
	for _, v := range h.GetAll("Set-Cookie") {
		j.ParseSetCookie(v)
	}
	return j
}
