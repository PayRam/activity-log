package useractivity

// ServiceMetadata is a structured metadata payload for service-level tracking.
type ServiceMetadata struct {
	Source         string                `json:"source"`
	OperationName  string                `json:"operationName,omitempty"`
	OperationTrail []OperationTrailEntry `json:"operationTrail,omitempty"`
}

func marshalServiceMetadata(metadata ServiceMetadata) *string {
	if len(metadata.OperationTrail) == 0 && metadata.OperationName == "" {
		return nil
	}
	return MergeMetadata(nil, metadata)
}

func mergeServiceMetadata(base *string, metadata ServiceMetadata) *string {
	if len(metadata.OperationTrail) == 0 && metadata.OperationName == "" {
		return base
	}
	return MergeMetadata(base, metadata)
}
