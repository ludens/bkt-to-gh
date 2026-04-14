package policy

import "testing"

func TestResolveVisibility(t *testing.T) {
	tests := []struct {
		name          string
		policy        VisibilityPolicy
		sourcePrivate bool
		want          bool
	}{
		{name: "all private forces private", policy: AllPrivate, sourcePrivate: false, want: true},
		{name: "all public forces public", policy: AllPublic, sourcePrivate: true, want: false},
		{name: "follow source keeps private", policy: FollowSource, sourcePrivate: true, want: true},
		{name: "follow source keeps public", policy: FollowSource, sourcePrivate: false, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveVisibility(tt.policy, tt.sourcePrivate)
			if err != nil {
				t.Fatalf("ResolveVisibility returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("ResolveVisibility() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveVisibilityRejectsUnknownPolicy(t *testing.T) {
	_, err := ResolveVisibility(VisibilityPolicy("bad"), true)
	if err == nil {
		t.Fatal("ResolveVisibility() error = nil, want error")
	}
}
