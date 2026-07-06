// Copyright (c) the go-ruby-http/http authors
//
// SPDX-License-Identifier: BSD-3-Clause

package http

import (
	"errors"
	"testing"
)

func TestErrorHierarchyIs(t *testing.T) {
	stateErr := newError(KindStateError, "boom")
	if !errors.Is(stateErr, ErrStateError) {
		t.Fatalf("StateError should match itself")
	}
	if !errors.Is(stateErr, ErrResponseError) {
		t.Fatalf("StateError should match ResponseError ancestor")
	}
	if !errors.Is(stateErr, ErrError) {
		t.Fatalf("every error should match ErrError root")
	}
	if errors.Is(stateErr, ErrTimeoutError) {
		t.Fatalf("StateError must not match TimeoutError")
	}
}

func TestErrorIsNonErrorTarget(t *testing.T) {
	e := newError(KindError, "x")
	if e.Is(errors.New("plain")) {
		t.Fatalf("must not match a non-*Error target")
	}
}

func TestErrorMessageAndUnwrap(t *testing.T) {
	cause := errors.New("dial failed")
	e := newTransportError(KindConnectionError, cause)
	if e.Error() != "dial failed" {
		t.Fatalf("message = %q", e.Error())
	}
	if !errors.Is(e, cause) {
		t.Fatalf("Unwrap should expose cause")
	}
	// nil cause path: message falls back to the kind string.
	e2 := newTransportError(KindTimeoutError, nil)
	if e2.Error() != string(KindTimeoutError) {
		t.Fatalf("nil-cause message = %q", e2.Error())
	}
	if e2.Unwrap() != nil {
		t.Fatalf("nil-cause unwrap not nil")
	}
}
