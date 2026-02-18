package repositories

// ProjectFilter is the enum for project ID filtering semantics.
type ProjectFilter string

const (
	ProjectFilterNoIDs ProjectFilter = "NO_IDS"
	ProjectFilterAll   ProjectFilter = "ALL"
)

// IsValid returns true when the filter value is one of the supported enums.
func (f ProjectFilter) IsValid() bool {
	switch f {
	case ProjectFilterNoIDs, ProjectFilterAll:
		return true
	default:
		return false
	}
}
