// Copyright (c) the go-ruby-http/http authors
//
// SPDX-License-Identifier: BSD-3-Clause

package http

import "strings"

// ContentType is a parsed Content-Type header, mirroring http.rb's
// HTTP::ContentType (a Struct of mime_type and charset).
type ContentType struct {
	// MimeType is the media type, stripped and lower-cased ("application/json"),
	// or "" when the header has no media type.
	MimeType string
	// Charset is the charset parameter value, stripped and unquoted, or "" when
	// absent.
	Charset string
}

// ParseContentType parses a Content-Type header value the way
// HTTP::ContentType.parse does: the media type is the text before the first ';',
// stripped and down-cased (empty ⇒ ""); the charset is the value of a
// case-insensitive `charset=` parameter, stripped and with surrounding quotes
// removed (absent ⇒ "").
func ParseContentType(value string) ContentType {
	return ContentType{
		MimeType: contentTypeMime(value),
		Charset:  contentTypeCharset(value),
	}
}

// contentTypeMime extracts the down-cased media type (before the first ';').
func contentTypeMime(value string) string {
	head := value
	if i := strings.IndexByte(head, ';'); i >= 0 {
		head = head[:i]
	}
	return strings.ToLower(strings.TrimSpace(head))
}

// contentTypeCharset extracts the charset parameter value (case-insensitive
// key), stripped and unquoted.
func contentTypeCharset(value string) string {
	for _, seg := range strings.Split(value, ";") {
		seg = strings.TrimSpace(seg)
		k, v, ok := strings.Cut(seg, "=")
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(k), "charset") {
			return strings.Trim(strings.TrimSpace(v), `"`)
		}
	}
	return ""
}
