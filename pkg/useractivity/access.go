package useractivity

import "context"

// AccessContext describes a member's access scope for user activity logs.
type AccessContext struct {
	IsAdmin           bool
	AllowedProjectIDs []uint
}

// AccessResolver resolves a member's access scope.
type AccessResolver interface {
	Resolve(ctx context.Context, memberID uint) (*AccessContext, error)
}

// ConfigProvider resolves integer configuration values by key.
type ConfigProvider interface {
	GetInt(ctx context.Context, key string) (int, bool, error)
}

// MemberResolver resolves member info by IDs.
type MemberResolver interface {
	GetByIDs(ctx context.Context, ids []uint) (map[uint]MemberInfo, error)
}

// ExternalPlatformResolver resolves external platform info by IDs.
type ExternalPlatformResolver interface {
	GetByIDs(ctx context.Context, ids []uint) (map[uint]ExternalPlatformInfo, error)
}
