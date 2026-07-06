// Copyright (c) the go-ruby-http/http authors
//
// SPDX-License-Identifier: BSD-3-Clause

package http

import "strings"

// KV is one ordered key/value header (or form/query) entry, the convenient
// literal for the variadic constructors.
type KV struct {
	Key string
	Val string
}

// Headers is http.rb's HTTP::Headers: an ordered, case-insensitive header map
// whose names are normalized to canonical HTTP header casing (see
// [NormalizeHeaderName]) and which may carry multiple values per name
// (Set-Cookie, …). Lookups are case-insensitive; [Headers.Get] returns the first
// value, [Headers.GetAll] every value in order.
type Headers struct {
	pairs []KV // normalized keys, insertion order, duplicates allowed
}

// NewHeaders builds a [Headers] from ordered entries (each added, so repeated
// keys accumulate as multiple values — use [Headers.Set] semantics via [KV] with
// distinct keys for single values).
func NewHeaders(kv ...KV) *Headers {
	h := &Headers{}
	for _, e := range kv {
		h.Add(e.Key, e.Val)
	}
	return h
}

// Len reports the number of stored header values (counting duplicates).
func (h *Headers) Len() int { return len(h.pairs) }

// Pairs returns the header entries in insertion order (normalized names). The
// slice must not be mutated.
func (h *Headers) Pairs() []KV { return h.pairs }

// Set replaces every value for key with a single value (HTTP::Headers#set / []=).
// The name is normalized; any existing occurrences are removed first.
func (h *Headers) Set(key, val string) {
	h.Delete(key)
	h.pairs = append(h.pairs, KV{Key: NormalizeHeaderName(key), Val: val})
}

// Add appends a value for key without removing existing ones
// (HTTP::Headers#add), so a name may hold several values.
func (h *Headers) Add(key, val string) {
	h.pairs = append(h.pairs, KV{Key: NormalizeHeaderName(key), Val: val})
}

// SetDefault sets key→val only when key is absent (case-insensitive).
func (h *Headers) SetDefault(key, val string) {
	if !h.Has(key) {
		h.Set(key, val)
	}
}

// Get returns the first value for key (case-insensitive) and whether present
// (HTTP::Headers#[]).
func (h *Headers) Get(key string) (string, bool) {
	nk := NormalizeHeaderName(key)
	for _, p := range h.pairs {
		if p.Key == nk {
			return p.Val, true
		}
	}
	return "", false
}

// GetAll returns every value for key in order (HTTP::Headers#get).
func (h *Headers) GetAll(key string) []string {
	nk := NormalizeHeaderName(key)
	var out []string
	for _, p := range h.pairs {
		if p.Key == nk {
			out = append(out, p.Val)
		}
	}
	return out
}

// Has reports whether key is present (case-insensitive).
func (h *Headers) Has(key string) bool {
	nk := NormalizeHeaderName(key)
	for _, p := range h.pairs {
		if p.Key == nk {
			return true
		}
	}
	return false
}

// Delete removes every value for key (case-insensitive), preserving order of the
// rest.
func (h *Headers) Delete(key string) {
	nk := NormalizeHeaderName(key)
	kept := h.pairs[:0:0]
	for _, p := range h.pairs {
		if p.Key != nk {
			kept = append(kept, p)
		}
	}
	h.pairs = kept
}

// Clone returns a deep copy of h.
func (h *Headers) Clone() *Headers {
	c := &Headers{pairs: make([]KV, len(h.pairs))}
	copy(c.pairs, h.pairs)
	return c
}

// Merge overlays other onto a copy of h: each of other's names replaces (via
// [Headers.Set]) the corresponding entry, so merged defaults are overridden.
func (h *Headers) Merge(other *Headers) *Headers {
	out := h.Clone()
	if other != nil {
		for _, p := range other.pairs {
			out.Set(p.Key, p.Val)
		}
	}
	return out
}

// NormalizeHeaderName canonicalizes a header name the way http.rb's
// HTTP::Headers::Normalizer does: split on '-' and '_', capitalize each part
// (first letter upper, the rest lower) and join with '-'. So "content_type" and
// "CONTENT-TYPE" both become "Content-Type", and "WWW-Authenticate" becomes
// "Www-Authenticate".
func NormalizeHeaderName(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool { return r == '-' || r == '_' })
	for i, p := range parts {
		parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
	}
	return strings.Join(parts, "-")
}
