// Copyright (c) the go-ruby-http/http authors
//
// SPDX-License-Identifier: BSD-3-Clause

package http

import "strings"

// Values is an ordered, multi-valued string map used for form bodies and query
// params, mirroring the Hashes http.rb threads into `form:` and `params:`.
// Insertion order is preserved (http.rb's FormData encoder keeps Hash order),
// and a key with several values is encoded array-style ("key[]=a&key[]=b"), as
// HTTP::FormData::Urlencoded does.
type Values struct {
	keys   []string
	values map[string][]string
	nilKey map[string]bool // keys added with a nil (bare) value
}

// NewValues builds a [Values] from ordered key/value entries (each added).
func NewValues(kv ...KV) *Values {
	v := &Values{values: map[string][]string{}, nilKey: map[string]bool{}}
	for _, e := range kv {
		v.Add(e.Key, e.Val)
	}
	return v
}

// ensure lazily initializes the maps for a zero-value [Values].
func (v *Values) ensure() {
	if v.values == nil {
		v.values = map[string][]string{}
		v.nilKey = map[string]bool{}
	}
}

// Add appends val under key, preserving insertion order. Repeated Adds under one
// key make it array-valued for encoding.
func (v *Values) Add(key, val string) *Values {
	v.ensure()
	if _, ok := v.values[key]; !ok {
		v.keys = append(v.keys, key)
	}
	v.values[key] = append(v.values[key], val)
	return v
}

// Set replaces any values under key with the single val.
func (v *Values) Set(key, val string) *Values {
	v.ensure()
	if _, ok := v.values[key]; !ok {
		v.keys = append(v.keys, key)
	}
	v.values[key] = []string{val}
	delete(v.nilKey, key)
	return v
}

// AddNil records a bare key with no value ("key" with no '='), matching a nil
// Hash value in http.rb's urlencoder.
func (v *Values) AddNil(key string) *Values {
	v.ensure()
	if _, ok := v.values[key]; !ok {
		v.keys = append(v.keys, key)
		v.values[key] = nil
	}
	v.nilKey[key] = true
	return v
}

// Get returns the first value for key and whether present.
func (v *Values) Get(key string) (string, bool) {
	vs, ok := v.values[key]
	if !ok || len(vs) == 0 {
		return "", false
	}
	return vs[0], true
}

// Len reports the number of distinct keys.
func (v *Values) Len() int { return len(v.keys) }

// Encode renders the values as an application/x-www-form-urlencoded string,
// byte-faithful to HTTP::FormData::Urlencoded: keys and values are escaped with
// [EscapeFormComponent] (space→'+', unreserved [A-Za-z0-9*-._] literal, else
// %XX), a multi-valued key is emitted array-style ("key[]=..."), and a bare
// (nil) key is emitted with no '='.
func (v *Values) Encode() string {
	var b strings.Builder
	first := true
	sep := func() {
		if !first {
			b.WriteByte('&')
		}
		first = false
	}
	for _, k := range v.keys {
		ek := EscapeFormComponent(k)
		vs := v.values[k]
		switch {
		case v.nilKey[k] && len(vs) == 0:
			sep()
			b.WriteString(ek)
		case len(vs) > 1:
			for _, val := range vs {
				sep()
				b.WriteString(ek)
				b.WriteString("[]=")
				b.WriteString(EscapeFormComponent(val))
			}
		default:
			sep()
			b.WriteString(ek)
			b.WriteByte('=')
			b.WriteString(EscapeFormComponent(vs[0]))
		}
	}
	return b.String()
}

// EscapeFormComponent percent-encodes s the way Ruby's
// URI.encode_www_form_component (used by HTTP::FormData) does: the set
// [A-Za-z0-9*-._] is left literal, a space becomes '+', and every other byte
// becomes %XX with upper-case hex.
func EscapeFormComponent(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == ' ':
			b.WriteByte('+')
		case formUnreserved(c):
			b.WriteByte(c)
		default:
			b.WriteByte('%')
			b.WriteByte(hexDigit(c >> 4))
			b.WriteByte(hexDigit(c & 0xf))
		}
	}
	return b.String()
}

// formUnreserved reports whether c is left literal by [EscapeFormComponent].
func formUnreserved(c byte) bool {
	switch {
	case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9':
		return true
	case c == '*', c == '-', c == '.', c == '_':
		return true
	}
	return false
}

// hexDigit maps a nibble (0..15) to its upper-case hexadecimal ASCII digit.
func hexDigit(n byte) byte {
	if n < 10 {
		return '0' + n
	}
	return 'A' + (n - 10)
}
