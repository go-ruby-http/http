// Copyright (c) the go-ruby-http/http authors
//
// SPDX-License-Identifier: BSD-3-Clause

package http

import "testing"

func TestStatusPredicates(t *testing.T) {
	cases := []struct {
		code                        int
		info, ok, redir, cerr, serr bool
	}{
		{100, true, false, false, false, false},
		{200, false, true, false, false, false},
		{301, false, false, true, false, false},
		{404, false, false, false, true, false},
		{500, false, false, false, false, true},
		{0, false, false, false, false, false},
		{600, false, false, false, false, false},
	}
	for _, c := range cases {
		s := Status(c.code)
		if s.Informational() != c.info || s.Success() != c.ok || s.Redirect() != c.redir ||
			s.ClientError() != c.cerr || s.ServerError() != c.serr {
			t.Fatalf("status %d predicates wrong", c.code)
		}
		if s.Code() != c.code {
			t.Fatalf("Code() = %d, want %d", s.Code(), c.code)
		}
	}
}

func TestStatusReasonAndString(t *testing.T) {
	if got := Status(200).Reason(); got != "OK" {
		t.Fatalf("reason = %q", got)
	}
	if got := Status(200).String(); got != "200 OK" {
		t.Fatalf("string = %q", got)
	}
	if got := Status(418).Reason(); got != "" {
		t.Fatalf("unknown reason = %q, want empty", got)
	}
	if got := Status(418).String(); got != "418" {
		t.Fatalf("unknown string = %q, want 418", got)
	}
	if got := Status(422).Reason(); got != "Unprocessable Entity" {
		t.Fatalf("422 = %q", got)
	}
}
