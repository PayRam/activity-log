package utils

import "encoding/json"

// MergeMetadata merges patch fields into existing metadata JSON.
// If existing is non-JSON, it is preserved under "rawMetadata".
func MergeMetadata(existing *string, patch interface{}) *string {
	if patch == nil {
		return existing
	}

	patchJSON, err := json.Marshal(patch)
	if err != nil {
		return existing
	}

	patchMap := map[string]interface{}{}
	if err := json.Unmarshal(patchJSON, &patchMap); err != nil || len(patchMap) == 0 {
		return existing
	}

	merged := map[string]interface{}{}
	if existing != nil && *existing != "" {
		if err := json.Unmarshal([]byte(*existing), &merged); err != nil {
			merged = map[string]interface{}{
				"rawMetadata": *existing,
			}
		}
	}

	for k, v := range patchMap {
		merged[k] = v
	}

	mergedJSON, err := json.Marshal(merged)
	if err != nil {
		return existing
	}

	jsonStr := string(mergedJSON)
	return &jsonStr
}
