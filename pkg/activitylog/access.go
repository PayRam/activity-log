package activitylog

import "context"

// AccessContext describes a member's access scope for activity log logs.
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

// ProjectResolver resolves project info by IDs.
type ProjectResolver interface {
	GetByIDs(ctx context.Context, ids []uint) (map[uint]ProjectInfo, error)
}

// Provider is a unified adapter for access, config, and hydration lookups.
// Implement this to avoid wiring separate resolver/provider interfaces.
type Provider interface {
	ResolveAccess(ctx context.Context, memberID uint) (*AccessContext, error)
	GetInt(ctx context.Context, key string) (int, bool, error)
	GetMembersByIDs(ctx context.Context, ids []uint) (map[uint]MemberInfo, error)
	GetProjectsByIDs(ctx context.Context, ids []uint) (map[uint]ProjectInfo, error)
}
