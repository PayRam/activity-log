package middleware

import (
	"mime"
	"strings"
)

func ShouldCaptureResponseBody(contentType, endpoint string, skipPathPrefixes []string) bool {
	for _, prefix := range skipPathPrefixes {
		if prefix != "" && strings.HasPrefix(endpoint, prefix) {
			return false
		}
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	}
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	if mediaType == "" {
		return false
	}

	if strings.HasPrefix(mediaType, "text/") {
		return true
	}

	if strings.HasSuffix(mediaType, "+json") || strings.HasSuffix(mediaType, "+xml") {
		return true
	}

	switch mediaType {
	case "application/json",
		"application/ld+json",
		"application/problem+json",
		"application/graphql-response+json",
		"application/xml",
		"application/javascript",
		"application/x-javascript",
		"application/x-www-form-urlencoded",
		"application/yaml",
		"application/x-yaml",
		"application/toml":
		return true
	}

	return false
}
