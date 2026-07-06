// Copyright (c) the go-ruby-http/http authors
//
// SPDX-License-Identifier: BSD-3-Clause

package http

import "strconv"

// Status is an HTTP status code with http.rb's HTTP::Response::Status query
// methods. The zero value is code 0 (a nil status, as http.rb models a response
// with no status line).
type Status int

// Code returns the numeric status code (HTTP::Response::Status#code / #to_i).
func (s Status) Code() int { return int(s) }

// Reason returns the reason phrase for the status code, or "" when the code has
// no registered phrase (HTTP::Response::Status#reason). The table mirrors
// HTTP::Response::Status::REASONS.
func (s Status) Reason() string { return statusReasons[int(s)] }

// String renders the status as http.rb's HTTP::Response::Status#to_s does:
// "<code> <reason>" when a reason phrase exists, otherwise just "<code>".
func (s Status) String() string {
	r := s.Reason()
	if r == "" {
		return strconv.Itoa(int(s))
	}
	return strconv.Itoa(int(s)) + " " + r
}

// Informational reports whether the code is 1xx (#informational?).
func (s Status) Informational() bool { return s >= 100 && s < 200 }

// Success reports whether the code is 2xx (#success?).
func (s Status) Success() bool { return s >= 200 && s < 300 }

// Redirect reports whether the code is 3xx (#redirect?).
func (s Status) Redirect() bool { return s >= 300 && s < 400 }

// ClientError reports whether the code is 4xx (#client_error?).
func (s Status) ClientError() bool { return s >= 400 && s < 500 }

// ServerError reports whether the code is 5xx (#server_error?).
func (s Status) ServerError() bool { return s >= 500 && s < 600 }

// statusReasons mirrors HTTP::Response::Status::REASONS (http gem 6.0.x).
var statusReasons = map[int]string{
	100: "Continue",
	101: "Switching Protocols",
	102: "Processing",
	200: "OK",
	201: "Created",
	202: "Accepted",
	203: "Non-Authoritative Information",
	204: "No Content",
	205: "Reset Content",
	206: "Partial Content",
	207: "Multi-Status",
	208: "Already Reported",
	226: "IM Used",
	300: "Multiple Choices",
	301: "Moved Permanently",
	302: "Found",
	303: "See Other",
	304: "Not Modified",
	305: "Use Proxy",
	307: "Temporary Redirect",
	308: "Permanent Redirect",
	400: "Bad Request",
	401: "Unauthorized",
	402: "Payment Required",
	403: "Forbidden",
	404: "Not Found",
	405: "Method Not Allowed",
	406: "Not Acceptable",
	407: "Proxy Authentication Required",
	408: "Request Timeout",
	409: "Conflict",
	410: "Gone",
	411: "Length Required",
	412: "Precondition Failed",
	413: "Payload Too Large",
	414: "URI Too Long",
	415: "Unsupported Media Type",
	416: "Range Not Satisfiable",
	417: "Expectation Failed",
	421: "Misdirected Request",
	422: "Unprocessable Entity",
	423: "Locked",
	424: "Failed Dependency",
	426: "Upgrade Required",
	428: "Precondition Required",
	429: "Too Many Requests",
	431: "Request Header Fields Too Large",
	451: "Unavailable For Legal Reasons",
	500: "Internal Server Error",
	501: "Not Implemented",
	502: "Bad Gateway",
	503: "Service Unavailable",
	504: "Gateway Timeout",
	505: "HTTP Version Not Supported",
	506: "Variant Also Negotiates",
	507: "Insufficient Storage",
	508: "Loop Detected",
	510: "Not Extended",
	511: "Network Authentication Required",
}
