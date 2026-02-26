package utils

import (
	"encoding/json"
	"strings"
	"unicode"
)

var defaultRedactKeys = []string{
	"password",
	"new_password",
	"old_password",
	"confirm_password",
	"token",
	"secret",
	"api_key",
	"apiKey",
	"client_secret",
	"clientSecret",
	"private_key",
	"privateKey",
	"access_token",
	"accessToken",
	"refresh_token",
	"refreshToken",
	"id_token",
	"idToken",
	"authorization",
	"Authorization",
	"passphrase",
	"passPhrase",
	"jwt",
	"bearer",
}

// DefaultRedactKeys returns a copy of built-in sensitive keys.
func DefaultRedactKeys() []string {
	return append([]string(nil), defaultRedactKeys...)
}

// RedactJSONKeys redacts the provided keys in a JSON payload.
// If payload is not valid JSON, it is returned unchanged.
func RedactJSONKeys(payload []byte, keys ...string) []byte {
	if len(payload) == 0 || len(keys) == 0 {
		return payload
	}

	redactSet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if key == "" {
			continue
		}
		normalized := normalizeRedactKey(key)
		if normalized == "" {
			continue
		}
		redactSet[normalized] = struct{}{}
	}
	if len(redactSet) == 0 {
		return payload
	}

	var data interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return payload
	}

	redactValue(data, redactSet)

	redacted, err := json.Marshal(data)
	if err != nil {
		return payload
	}
	return redacted
}

func redactValue(value interface{}, keys map[string]struct{}) {
	switch v := value.(type) {
	case map[string]interface{}:
		for k, val := range v {
			if _, ok := keys[normalizeRedactKey(k)]; ok {
				v[k] = "***REDACTED***"
				continue
			}
			redactValue(val, keys)
		}
	case []interface{}:
		for i := range v {
			redactValue(v[i], keys)
		}
	}
}

func normalizeRedactKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(key))
	for _, r := range key {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}
