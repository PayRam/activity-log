package middleware

import "testing"

func TestShouldCaptureResponseBodyContentTypes(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        bool
	}{
		{name: "empty content type", contentType: "", want: false},
		{name: "whitespace content type", contentType: "   ", want: false},
		{name: "invalid content type", contentType: ";;; ", want: false},
		{name: "json content type", contentType: "application/json; charset=utf-8", want: true},
		{name: "vendor json content type", contentType: "application/vnd.api+json", want: true},
		{name: "problem json content type", contentType: "application/problem+json", want: true},
		{name: "xml content type", contentType: "application/xml", want: true},
		{name: "vendor xml content type", contentType: "application/atom+xml", want: true},
		{name: "text content type", contentType: "text/plain", want: true},
		{name: "png content type", contentType: "image/png", want: false},
		{name: "octet stream content type", contentType: "application/octet-stream", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldCaptureResponseBody(tt.contentType, "/endpoint", nil)
			if got != tt.want {
				t.Fatalf("ShouldCaptureResponseBody(%q) = %v, want %v", tt.contentType, got, tt.want)
			}
		})
	}
}

func TestShouldCaptureResponseBodySkipsConfiguredPrefixes(t *testing.T) {
	if ShouldCaptureResponseBody("application/json", "/internal/health", []string{"/internal"}) {
		t.Fatalf("expected configured path prefix to skip body capture")
	}
}
