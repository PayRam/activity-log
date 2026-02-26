package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestNormalizeMaxBodyBytes(t *testing.T) {
	if got := NormalizeMaxBodyBytes(0); got != defaultMaxBodyBytes {
		t.Fatalf("expected default %d, got %d", defaultMaxBodyBytes, got)
	}
	if got := NormalizeMaxBodyBytes(64); got != 64 {
		t.Fatalf("expected 64, got %d", got)
	}
}

func TestMethodToAction(t *testing.T) {
	cases := map[string]string{
		"GET":    "READ",
		"post":   "WRITE",
		"PUT":    "WRITE",
		"PATCH":  "WRITE",
		"DELETE": "DELETE",
		"TRACE":  "UNKNOWN",
	}
	for method, expected := range cases {
		if got := MethodToAction(method); got != expected {
			t.Fatalf("method %s: expected %s, got %s", method, expected, got)
		}
	}
}

func TestStatusToAPIStatus(t *testing.T) {
	if got := StatusToAPIStatus(200); got != "SUCCESS" {
		t.Fatalf("expected SUCCESS, got %s", got)
	}
	if got := StatusToAPIStatus(404); got != "DENIED" {
		t.Fatalf("expected DENIED, got %s", got)
	}
	if got := StatusToAPIStatus(500); got != "ERROR" {
		t.Fatalf("expected ERROR, got %s", got)
	}
}

func TestReadRequestBody(t *testing.T) {
	req := &http.Request{Body: io.NopCloser(strings.NewReader("hello"))}
	body, err := ReadRequestBody(req, 10, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body == nil || *body != "hello" {
		t.Fatalf("expected body 'hello', got %#v", body)
	}

	// Ensure body is restored
	restored, _ := io.ReadAll(req.Body)
	if string(restored) != "hello" {
		t.Fatalf("expected restored body 'hello', got %q", string(restored))
	}

	req = &http.Request{Body: io.NopCloser(strings.NewReader("secret"))}
	body, err = ReadRequestBody(req, 10, func(b []byte) []byte {
		return []byte("redacted")
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body == nil || *body != "redacted" {
		t.Fatalf("expected redacted body, got %#v", body)
	}

	req = &http.Request{Body: io.NopCloser(strings.NewReader("abcdef"))}
	body, err = ReadRequestBody(req, 3, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body == nil || *body != "abc" {
		t.Fatalf("expected truncated body 'abc', got %#v", body)
	}
}

func TestBodyCapture(t *testing.T) {
	capture := NewBodyCapture(5)
	capture.Write([]byte("hello"))
	if got := capture.String(); got != "hello" {
		t.Fatalf("expected hello, got %q", got)
	}

	capture = NewBodyCapture(5)
	capture.Write([]byte("hello world"))
	if got := capture.String(); got != "hello..." {
		t.Fatalf("expected truncated body, got %q", got)
	}

	capture = NewBodyCapture(3)
	capture.Write([]byte("ab"))
	capture.Write([]byte("cd"))
	if got := capture.String(); got != "abc..." {
		t.Fatalf("expected truncated body abc..., got %q", got)
	}

	// Ensure no panic on nil
	var nilCapture *BodyCapture
	nilCapture.Write([]byte("x"))
	if got := nilCapture.String(); got != "" {
		t.Fatalf("expected empty string for nil capture, got %q", got)
	}

	// Ensure buffer doesn't grow beyond max
	capture = NewBodyCapture(2)
	capture.Write(bytes.Repeat([]byte("x"), 10))
	if got := capture.String(); got != "xx..." {
		t.Fatalf("expected xx..., got %q", got)
	}
}
