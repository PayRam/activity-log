package utils

import (
	"sort"

	"github.com/PayRam/activity-log/internal/models"
)

// CollectMemberIDs returns a distinct member ID list.
func CollectMemberIDs(list []models.ActivityLog) []uint {
	idSet := make(map[uint]struct{})
	for _, activity := range list {
		if activity.MemberID != nil {
			idSet[*activity.MemberID] = struct{}{}
		}
	}
	return idMapToSlice(idSet)
}

// CollectExternalPlatformIDs returns a distinct external platform ID list.
func CollectExternalPlatformIDs(list []models.ActivityLog) []uint {
	idSet := make(map[uint]struct{})
	for _, activity := range list {
		for _, id := range activity.ExternalPlatformIDs {
			idSet[id] = struct{}{}
		}
	}
	return idMapToSlice(idSet)
}

func idMapToSlice(idSet map[uint]struct{}) []uint {
	if len(idSet) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
