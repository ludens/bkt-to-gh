package policy

import "fmt"

type VisibilityPolicy string

const (
	AllPrivate   VisibilityPolicy = "all-private"
	AllPublic    VisibilityPolicy = "all-public"
	FollowSource VisibilityPolicy = "follow-source"
)

func ResolveVisibility(policy VisibilityPolicy, sourcePrivate bool) (bool, error) {
	switch policy {
	case AllPrivate:
		return true, nil
	case AllPublic:
		return false, nil
	case FollowSource:
		return sourcePrivate, nil
	default:
		return false, fmt.Errorf("unknown visibility policy %q", policy)
	}
}
