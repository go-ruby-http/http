// Copyright (c) the go-ruby-http/http authors
//
// SPDX-License-Identifier: BSD-3-Clause

package http

// Error is the root of http.rb's error tree (HTTP::Error). Every error carries a
// message and a [Error.Kind] naming the concrete HTTP:: subclass; a response
// error also carries the [Error.Response] that triggered it, and a transport
// error wraps the underlying cause. Match with errors.Is against the Err*
// sentinels — a superclass sentinel matches its subclasses, mirroring Ruby's
// rescue of a superclass.
type Error struct {
	// Kind names the HTTP:: error subclass (see the Err* sentinels).
	Kind ErrorKind
	// Message is the error text (Exception#message).
	Message string
	// Response is the response context for a ResponseError (nil otherwise).
	Response *Response
	// Cause is the underlying transport error, if any.
	Cause error
}

// ErrorKind identifies an http.rb error subclass.
type ErrorKind string

// The http.rb error subclasses, named as in the gem (HTTP::*).
const (
	KindError           ErrorKind = "HTTP::Error"
	KindConnectionError ErrorKind = "HTTP::ConnectionError"
	KindRequestError    ErrorKind = "HTTP::RequestError"
	KindResponseError   ErrorKind = "HTTP::ResponseError"
	KindStateError      ErrorKind = "HTTP::StateError"
	KindTimeoutError    ErrorKind = "HTTP::TimeoutError"
	KindHeaderError     ErrorKind = "HTTP::HeaderError"
)

// Sentinel errors for errors.Is matching. A concrete [Error] whose Kind is the
// sentinel's Kind (or a descendant of it) matches via [Error.Is].
var (
	ErrError           = &Error{Kind: KindError, Message: string(KindError)}
	ErrConnectionError = &Error{Kind: KindConnectionError, Message: string(KindConnectionError)}
	ErrRequestError    = &Error{Kind: KindRequestError, Message: string(KindRequestError)}
	ErrResponseError   = &Error{Kind: KindResponseError, Message: string(KindResponseError)}
	ErrStateError      = &Error{Kind: KindStateError, Message: string(KindStateError)}
	ErrTimeoutError    = &Error{Kind: KindTimeoutError, Message: string(KindTimeoutError)}
	ErrHeaderError     = &Error{Kind: KindHeaderError, Message: string(KindHeaderError)}
)

// errorParents maps each kind to its parent in the http.rb hierarchy:
//
//	HTTP::Error
//	├── HTTP::ConnectionError
//	├── HTTP::RequestError
//	├── HTTP::ResponseError
//	│   └── HTTP::StateError
//	├── HTTP::TimeoutError
//	└── HTTP::HeaderError
var errorParents = map[ErrorKind]ErrorKind{
	KindConnectionError: KindError,
	KindRequestError:    KindError,
	KindResponseError:   KindError,
	KindTimeoutError:    KindError,
	KindHeaderError:     KindError,
	KindStateError:      KindResponseError,
}

// Error implements the error interface (Exception#message).
func (e *Error) Error() string { return e.Message }

// Unwrap exposes the underlying transport cause for errors.Is/As.
func (e *Error) Unwrap() error { return e.Cause }

// Is reports whether e matches target: true when target is a [*Error] whose Kind
// is e's Kind or an ancestor of it, so errors.Is(err, ErrResponseError) matches a
// StateError and errors.Is(err, ErrError) matches every http.rb error.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	for k := e.Kind; ; {
		if k == t.Kind {
			return true
		}
		parent, ok := errorParents[k]
		if !ok {
			return false
		}
		k = parent
	}
}

// newError builds an [Error] of the given kind with a message.
func newError(kind ErrorKind, msg string) *Error {
	return &Error{Kind: kind, Message: msg}
}

// newTransportError builds a transport [Error] (ConnectionError / TimeoutError)
// wrapping the adapter's underlying error as the cause.
func newTransportError(kind ErrorKind, cause error) *Error {
	msg := string(kind)
	if cause != nil {
		msg = cause.Error()
	}
	return &Error{Kind: kind, Message: msg, Cause: cause}
}
