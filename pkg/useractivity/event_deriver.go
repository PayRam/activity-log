package useractivity

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"unicode"
)

const defaultEventDeriverBasePath = "/api/v1"

// EventDeriverInput contains data used to derive event values.
type EventDeriverInput struct {
	Endpoint    string
	Method      string
	RequestBody *string
	StatusCode  *HTTPStatusCode
	APIStatus   APIStatus
}

// EventDeriver derives (eventCategory, eventName) from request context.
// It is used only when CreateRequest.EventCategory and/or CreateRequest.EventName are missing.
type EventDeriver func(input EventDeriverInput) (eventCategory, eventName string)

// EventInfo contains derived event fields.
type EventInfo struct {
	EventCategory string
	EventName     string
	Description   string
}

// EventInfoDeriver derives event category, event name, and description from request context.
// It is used only when CreateRequest/UpdateRequest fields are missing.
type EventInfoDeriver func(input EventDeriverInput) EventInfo

// CoreLikeEventDeriverConfig configures NewCoreLikeEventDeriver.
// This is useful when you want behavior similar to test/core deriveEventInfo.
type CoreLikeEventDeriverConfig struct {
	BasePath   string
	TableNames []string
}

// DefaultEventDeriver derives both values from the endpoint segment after /api/v1/.
// Example: /api/v1/payment-request -> payment-request, payment-request.
func DefaultEventDeriver(input EventDeriverInput) (string, string) {
	name := deriveEventNameFromEndpoint(input.Endpoint)
	if name == "" {
		return "", ""
	}
	return name, name
}

// DefaultEventInfoDeriver derives category, name, and description from endpoint/method/status.
func DefaultEventInfoDeriver(input EventDeriverInput) EventInfo {
	eventCategory, eventName := DefaultEventDeriver(input)
	return EventInfo{
		EventCategory: eventCategory,
		EventName:     eventName,
		Description:   buildEventDescription(input, eventName, deriveEventActionFromEndpoint(input.Endpoint, input.Method)),
	}
}

// NewCoreLikeEventDeriver returns an event deriver that approximates test/core deriveEventInfo behavior:
// - Category: matched table name upper-cased (or fallback to normalized resource segment)
// - Name: CATEGORY_ACTION where action is derived from HTTP method
func NewCoreLikeEventDeriver(cfg CoreLikeEventDeriverConfig) EventDeriver {
	infoDeriver := NewCoreLikeEventInfoDeriver(cfg)
	return func(input EventDeriverInput) (string, string) {
		info := infoDeriver(input)
		return info.EventCategory, info.EventName
	}
}

// NewCoreLikeEventInfoDeriver returns an event info deriver that approximates test/core deriveEventInfo behavior:
// - Category: matched table name upper-cased (or fallback to normalized resource segment)
// - Name: CATEGORY_ACTION where action is derived from HTTP method
// - Description: method/status-aware message with optional sanitized request-body preview
func NewCoreLikeEventInfoDeriver(cfg CoreLikeEventDeriverConfig) EventInfoDeriver {
	basePath := strings.TrimSpace(cfg.BasePath)
	if basePath == "" {
		basePath = defaultEventDeriverBasePath
	}
	baseParts := normalizedPathSegments(basePath)
	tables := append([]string(nil), cfg.TableNames...)

	return func(input EventDeriverInput) EventInfo {
		segments := normalizedPathSegments(input.Endpoint)
		if len(segments) == 0 {
			return EventInfo{}
		}

		resourceParts := segments
		if len(baseParts) > 0 && hasPrefixSegments(segments, baseParts) && len(segments) > len(baseParts) {
			resourceParts = segments[len(baseParts):]
		}
		if len(resourceParts) == 0 {
			return EventInfo{}
		}

		category := ""
		resource := ""
		if table := findLastMatchingTable(resourceParts, tables); table != "" {
			category = strings.ToUpper(normalizeEventToken(table))
			resource = table
		} else {
			resource = resourceParts[0]
			category = strings.ToUpper(normalizeEventToken(resource))
		}
		if category == "" {
			return EventInfo{}
		}

		action := deriveActionFromMethod(input.Method, resourceParts, resource)
		if action == "" {
			action = "UNKNOWN"
		}
		return EventInfo{
			EventCategory: category,
			EventName:     category + "_" + action,
			Description:   buildEventDescription(input, resource, action),
		}
	}
}

func deriveEventInfo(input EventDeriverInput, eventDeriver EventDeriver, eventInfoDeriver EventInfoDeriver) EventInfo {
	if eventDeriver == nil && eventInfoDeriver == nil {
		return DefaultEventInfoDeriver(input)
	}

	info := EventInfo{}
	if eventInfoDeriver != nil {
		info = eventInfoDeriver(input)
	}

	if info.EventCategory == "" || info.EventName == "" {
		if eventDeriver == nil {
			eventDeriver = DefaultEventDeriver
		}
		eventCategory, eventName := eventDeriver(input)
		if info.EventCategory == "" {
			info.EventCategory = eventCategory
		}
		if info.EventName == "" {
			info.EventName = eventName
		}
	}

	if info.EventCategory == "" {
		info.EventCategory = info.EventName
	}
	if info.Description == "" {
		resource := info.EventName
		if resource == "" {
			resource = deriveEventNameFromEndpoint(input.Endpoint)
		}
		info.Description = buildEventDescription(input, resource, deriveEventActionFromEndpoint(input.Endpoint, input.Method))
	}

	return info
}

func deriveEventNameFromEndpoint(endpoint string) string {
	path := strings.TrimSpace(endpoint)
	if path == "" {
		return ""
	}

	segments := normalizedPathSegments(path)
	if len(segments) == 0 {
		return ""
	}

	for i := 0; i+2 < len(segments); i++ {
		if segments[i] == "api" && strings.HasPrefix(segments[i+1], "v") {
			return segments[i+2]
		}
	}

	return segments[0]
}

func deriveEventActionFromEndpoint(endpoint, method string) string {
	resourceParts := normalizedPathSegments(endpoint)
	for i := 0; i+2 < len(resourceParts); i++ {
		if resourceParts[i] == "api" && strings.HasPrefix(resourceParts[i+1], "v") {
			resourceParts = resourceParts[i+2:]
			break
		}
	}

	resource := ""
	if len(resourceParts) > 0 {
		resource = resourceParts[0]
	}
	return deriveActionFromMethod(method, resourceParts, resource)
}

func normalizedPathSegments(path string) []string {
	if path == "" {
		return nil
	}
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	raw := strings.Split(strings.Trim(path, "/"), "/")
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, strings.ToLower(s))
	}
	return out
}

func hasPrefixSegments(segments, prefix []string) bool {
	if len(prefix) == 0 || len(segments) < len(prefix) {
		return false
	}
	for i := range prefix {
		if segments[i] != prefix[i] {
			return false
		}
	}
	return true
}

func findLastMatchingTable(segments []string, tableNames []string) string {
	if len(segments) == 0 || len(tableNames) == 0 {
		return ""
	}

	lastMatch := ""
	for _, segment := range segments {
		segNormalized := normalizeEventToken(segment)
		segVariants := singularPluralVariants(segNormalized)
		for _, table := range tableNames {
			tableNormalized := normalizeEventToken(table)
			if tableNormalized == "" {
				continue
			}
			for _, variant := range segVariants {
				if variant == tableNormalized {
					lastMatch = table
					break
				}
			}
		}
	}
	return lastMatch
}

func singularPluralVariants(name string) []string {
	if name == "" {
		return nil
	}

	variants := []string{name}
	if strings.HasSuffix(name, "ies") {
		variants = append(variants, strings.TrimSuffix(name, "ies")+"y")
	}
	if strings.HasSuffix(name, "y") {
		variants = append(variants, strings.TrimSuffix(name, "y")+"ies")
	}
	if strings.HasSuffix(name, "s") {
		variants = append(variants, strings.TrimSuffix(name, "s"))
	} else {
		variants = append(variants, name+"s")
	}
	return variants
}

func deriveActionFromMethod(method string, resourceParts []string, resource string) string {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "GET":
		if len(resourceParts) <= 1 {
			return "LIST"
		}
		last := normalizeEventToken(resourceParts[len(resourceParts)-1])
		base := normalizeEventToken(resource)
		if last == "" {
			return "LIST"
		}
		if strings.HasPrefix(last, ":") {
			return "VIEW"
		}
		for _, variant := range singularPluralVariants(base) {
			if last == variant {
				return "LIST"
			}
		}
		if looksLikeIdentifier(last) {
			return "VIEW"
		}
		return "VIEW"
	case "POST":
		return "CREATE"
	case "PUT", "PATCH":
		return "UPDATE"
	case "DELETE":
		return "DELETE"
	default:
		method = strings.ToUpper(strings.TrimSpace(method))
		if method == "" {
			return "UNKNOWN"
		}
		return method
	}
}

func looksLikeIdentifier(value string) bool {
	if value == "" {
		return false
	}
	allDigits := true
	for _, r := range value {
		if !unicode.IsDigit(r) {
			allDigits = false
			break
		}
	}
	if allDigits {
		return true
	}
	if strings.Contains(value, "-") && len(value) >= 8 {
		return true
	}
	return false
}

func normalizeEventToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func buildEventDescription(input EventDeriverInput, rawResource, action string) string {
	resource := humanizeResourceName(rawResource)
	if resource == "" {
		resource = "resource"
	}

	isSuccess := isSuccessStatus(resolveStatusCode(input.StatusCode, input.APIStatus))
	switch action {
	case "LIST":
		if isSuccess {
			return fmt.Sprintf("Successfully retrieved %s list", resource)
		}
		return fmt.Sprintf("Failed to retrieve %s list", resource)
	case "VIEW":
		if isSuccess {
			return fmt.Sprintf("Successfully retrieved %s", resource)
		}
		return fmt.Sprintf("Failed to retrieve %s", resource)
	case "CREATE":
		if isSuccess {
			if preview := extractBodyPreview(input.RequestBody, 100); preview != "" {
				return fmt.Sprintf("Successfully created %s with values: %s", resource, preview)
			}
			return fmt.Sprintf("Successfully created %s", resource)
		}
		return fmt.Sprintf("Failed to create %s", resource)
	case "UPDATE":
		if isSuccess {
			if preview := extractBodyPreview(input.RequestBody, 100); preview != "" {
				return fmt.Sprintf("Successfully updated %s with values: %s", resource, preview)
			}
			return fmt.Sprintf("Successfully updated %s", resource)
		}
		return fmt.Sprintf("Failed to update %s", resource)
	case "DELETE":
		if isSuccess {
			return fmt.Sprintf("Successfully deleted %s", resource)
		}
		return fmt.Sprintf("Failed to delete %s", resource)
	default:
		method := strings.ToUpper(strings.TrimSpace(input.Method))
		if method == "" {
			method = "OPERATION"
		}
		if isSuccess {
			return fmt.Sprintf("Successfully performed %s on %s", method, resource)
		}
		return fmt.Sprintf("Failed to perform %s on %s", method, resource)
	}
}

func resolveStatusCode(statusCode *HTTPStatusCode, apiStatus APIStatus) int {
	if statusCode != nil {
		return int(*statusCode)
	}

	switch APIStatus(strings.ToUpper(strings.TrimSpace(string(apiStatus)))) {
	case APIStatusError:
		return http.StatusInternalServerError
	case APIStatusDenied:
		return http.StatusForbidden
	case APIStatusSuccess:
		return http.StatusOK
	default:
		return http.StatusOK
	}
}

func isSuccessStatus(status int) bool {
	return status >= 200 && status < 300
}

func humanizeResourceName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.TrimPrefix(value, "/")
	if idx := strings.LastIndex(value, "/"); idx >= 0 && idx+1 < len(value) {
		value = value[idx+1:]
	}
	value = normalizeEventToken(value)
	value = strings.TrimSuffix(value, "_list")
	value = strings.TrimSuffix(value, "_view")
	value = strings.TrimSuffix(value, "_create")
	value = strings.TrimSuffix(value, "_update")
	value = strings.TrimSuffix(value, "_delete")

	if strings.HasSuffix(value, "ies") {
		value = strings.TrimSuffix(value, "ies") + "y"
	} else if strings.HasSuffix(value, "s") {
		value = strings.TrimSuffix(value, "s")
	}

	value = strings.ReplaceAll(value, "_", " ")
	return strings.TrimSpace(value)
}

func extractBodyPreview(body *string, maxLen int) string {
	if body == nil || strings.TrimSpace(*body) == "" {
		return ""
	}

	var bodyMap map[string]interface{}
	if err := json.Unmarshal([]byte(*body), &bodyMap); err != nil {
		return ""
	}

	keys := make([]string, 0, len(bodyMap))
	for key := range bodyMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	preview := make([]string, 0, 3)
	for _, key := range keys {
		if isSensitiveBodyField(key) {
			continue
		}
		preview = append(preview, fmt.Sprintf("%s=%v", key, bodyMap[key]))
		if len(preview) >= 3 {
			break
		}
	}

	if len(preview) == 0 {
		return ""
	}

	result := strings.Join(preview, ", ")
	if len(result) > maxLen {
		result = result[:maxLen] + "..."
	}
	return result
}

func isSensitiveBodyField(key string) bool {
	key = normalizeBodyFieldKey(key)
	if key == "" {
		return false
	}

	knownSensitive := map[string]bool{
		"password":        true,
		"newpassword":     true,
		"oldpassword":     true,
		"confirmpassword": true,
		"token":           true,
		"accesstoken":     true,
		"refreshtoken":    true,
		"idtoken":         true,
		"authorization":   true,
		"apikey":          true,
		"clientsecret":    true,
		"secret":          true,
		"privatekey":      true,
		"jwt":             true,
		"bearer":          true,
	}

	if knownSensitive[key] {
		return true
	}
	return strings.Contains(key, "password") || strings.HasSuffix(key, "token") || strings.Contains(key, "secret")
}

func normalizeBodyFieldKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.ReplaceAll(key, "_", "")
	key = strings.ReplaceAll(key, "-", "")
	key = strings.ReplaceAll(key, " ", "")
	return key
}
