// Copyright (c) the go-ruby-http/http authors
//
// SPDX-License-Identifier: BSD-3-Clause

package http

import (
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// The oracle tests diff this package against the reference `http` gem (http.rb):
// they drive the gem's HTTP::Headers::Normalizer, HTTP::FormData::Urlencoded,
// HTTP::ContentType, HTTP::Response::Status and Basic-auth building, and assert
// byte-for-byte agreement. They skip themselves where the gem (or ruby) is
// absent — the qemu cross-arch and Windows lanes — so the deterministic,
// ruby-free suite alone holds the 100% coverage gate there.

// gemRuby returns a ruby whose `http` gem exposes the helpers we diff against, or
// skips the test.
func gemRuby(t *testing.T) string {
	t.Helper()
	bin, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not on PATH; skipping http-gem oracle")
	}
	probe := `require "http"
require "http/form_data"
need = defined?(HTTP::Headers::Normalizer) &&
       defined?(HTTP::FormData::Urlencoded) &&
       defined?(HTTP::ContentType) &&
       defined?(HTTP::Response::Status)
exit(need ? 0 : 1)`
	if err := exec.Command(bin, "-e", probe).Run(); err != nil {
		t.Skip("http gem absent or too old for the oracle; skipping")
	}
	return bin
}

// rubyEval runs a ruby script (http required, stdout binary) and returns the
// newline-trimmed stdout, failing on error.
func rubyEval(t *testing.T, bin, script string) string {
	t.Helper()
	cmd := exec.Command(bin, "-rhttp", "-rhttp/form_data", "-e", "$stdout.binmode\n"+script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ruby error: %v\nscript:\n%s\noutput:\n%s", err, script, out)
	}
	return strings.TrimRight(string(out), "\n")
}

// rubyString renders a Go string as a ruby double-quoted literal with \xNN
// escapes, so arbitrary bytes survive the shell and the ruby parser.
func rubyString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		b.WriteString("\\x")
		b.WriteByte("0123456789ABCDEF"[s[i]>>4])
		b.WriteByte("0123456789ABCDEF"[s[i]&0xf])
	}
	b.WriteByte('"')
	return b.String()
}

func TestOracleNormalizeHeaderName(t *testing.T) {
	bin := gemRuby(t)
	for _, s := range []string{"content-type", "CONTENT_TYPE", "WWW-Authenticate", "ETag", "x-custom-header"} {
		want := rubyEval(t, bin, "print HTTP::Headers::Normalizer.new.call("+rubyString(s)+")")
		if got := NormalizeHeaderName(s); got != want {
			t.Fatalf("NormalizeHeaderName(%q) = %q, gem = %q", s, got, want)
		}
	}
}

func TestOracleUrlencoded(t *testing.T) {
	bin := gemRuby(t)
	v := NewValues(KV{"a", "1"}).Add("b", "x y").Add("c", "1").Add("c", "2")
	want := rubyEval(t, bin,
		`print HTTP::FormData::Urlencoded.new({"a"=>"1","b"=>"x y","c"=>["1","2"]}).to_s`)
	if got := v.Encode(); got != want {
		t.Fatalf("Values.Encode = %q, gem = %q", got, want)
	}
}

func TestOracleEscapeFormComponent(t *testing.T) {
	bin := gemRuby(t)
	for _, s := range []string{"~-_.!* /a", "é&x=y", "plain*.-_"} {
		want := rubyEval(t, bin, "print URI.encode_www_form_component("+rubyString(s)+")")
		if got := EscapeFormComponent(s); got != want {
			t.Fatalf("EscapeFormComponent(%q) = %q, gem = %q", s, got, want)
		}
	}
}

func TestOracleContentType(t *testing.T) {
	bin := gemRuby(t)
	for _, s := range []string{
		"application/json; charset=utf-8", "text/html", `TEXT/HTML; Charset="UTF-8"`, "  application/JSON  ",
	} {
		want := rubyEval(t, bin, `ct = HTTP::ContentType.parse(`+rubyString(s)+`)
print [ct.mime_type, ct.charset].inspect`)
		ct := ParseContentType(s)
		got := "[" + rubyInspect(ct.MimeType) + ", " + rubyInspect(ct.Charset) + "]"
		if got != want {
			t.Fatalf("ParseContentType(%q) = %s, gem = %s", s, got, want)
		}
	}
}

func TestOracleStatusReasons(t *testing.T) {
	bin := gemRuby(t)
	for _, code := range []int{100, 200, 201, 204, 301, 404, 418, 422, 429, 500, 503} {
		want := rubyEval(t, bin, "print HTTP::Response::Status.new("+strconv.Itoa(code)+").reason.inspect")
		got := rubyInspect(Status(code).Reason())
		if got != want {
			t.Fatalf("Status(%d).Reason = %s, gem = %s", code, got, want)
		}
	}
}

func TestOracleStatusToS(t *testing.T) {
	bin := gemRuby(t)
	for _, code := range []int{200, 404, 418} {
		want := rubyEval(t, bin, "print HTTP::Response::Status.new("+strconv.Itoa(code)+").to_s")
		if got := Status(code).String(); got != want {
			t.Fatalf("Status(%d).String = %q, gem = %q", code, got, want)
		}
	}
}

func TestOracleBasicAuth(t *testing.T) {
	bin := gemRuby(t)
	want := rubyEval(t, bin, `print HTTP.basic_auth(user: "aladdin", pass: "opensesame").default_options.headers["Authorization"]`)
	if got := BasicAuthHeader("aladdin", "opensesame"); got != want {
		t.Fatalf("BasicAuthHeader = %q, gem = %q", got, want)
	}
}

// rubyInspect renders a Go string the way Ruby's String#inspect / nil.inspect
// would for the empty-vs-nil distinction the ContentType/Status oracles compare:
// "" ⇒ Ruby nil ("nil"), otherwise a quoted literal.
func rubyInspect(s string) string {
	if s == "" {
		return "nil"
	}
	return `"` + s + `"`
}
