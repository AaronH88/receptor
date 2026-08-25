package services

import (
	"testing"

	"github.com/ansible/receptor/pkg/netceptor"
)

func TestTagsMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		adTags   map[string]string
		selector map[string]string
		want     bool
	}{
		{
			name:     "empty selector matches anything",
			adTags:   map[string]string{"region": "us-east-1"},
			selector: map[string]string{},
			want:     true,
		},
		{
			name:     "single key match",
			adTags:   map[string]string{"kind": "script", "region": "us-east-1"},
			selector: map[string]string{"kind": "script"},
			want:     true,
		},
		{
			name:     "all keys must match",
			adTags:   map[string]string{"kind": "script", "gpu": "false"},
			selector: map[string]string{"kind": "script", "gpu": "false"},
			want:     true,
		},
		{
			name:     "missing key",
			adTags:   map[string]string{"kind": "script"},
			selector: map[string]string{"gpu": "true"},
			want:     false,
		},
		{
			name:     "value mismatch",
			adTags:   map[string]string{"region": "us-west-2"},
			selector: map[string]string{"region": "us-east-1"},
			want:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := TagsMatch(tc.adTags, tc.selector); got != tc.want {
				t.Fatalf("TagsMatch() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDiscoverServicesByTags(t *testing.T) {
	t.Parallel()

	ads := []*netceptor.ServiceAdvertisement{
		{NodeID: "bridge-east", Service: "scr", Tags: map[string]string{"kind": "script", "region": "us-east-1"}},
		{NodeID: "bridge-west", Service: "scr", Tags: map[string]string{"kind": "script", "region": "us-west-2"}},
		{NodeID: "bridge-east", Service: "agt", Tags: map[string]string{"kind": "agent", "region": "us-east-1"}},
		{NodeID: "worker", Service: "lease", Tags: map[string]string{"type": "Worker Node", "pool": "script"}},
	}

	got := DiscoverServicesByTags(ads, map[string]string{"kind": "script"}, "", "")
	if len(got) != 2 {
		t.Fatalf("expected 2 script endpoints, got %d", len(got))
	}

	got = DiscoverServicesByTags(ads, map[string]string{"kind": "script", "region": "us-east-1"}, "", "")
	if len(got) != 1 || got[0].NodeID != "bridge-east" || got[0].Service != "scr" {
		t.Fatalf("unexpected east script endpoint: %+v", got)
	}

	got = DiscoverServicesByTags(ads, map[string]string{"pool": "script"}, "", "")
	if len(got) != 1 || got[0].Service != "lease" {
		t.Fatalf("expected lease service match, got %+v", got)
	}

	got = DiscoverServicesByTags(ads, map[string]string{"kind": "script"}, "bridge-east", "scr")
	if len(got) != 1 || got[0].NodeID != "bridge-east" {
		t.Fatalf("expected node filter to apply, got %+v", got)
	}
}

func TestRoundRobinPicker(t *testing.T) {
	t.Parallel()

	choices := []ServiceEndpoint{
		{NodeID: "a", Service: "one"},
		{NodeID: "b", Service: "two"},
		{NodeID: "c", Service: "three"},
	}
	rr := RoundRobinPicker{}

	order := make([]string, 6)
	for i := range order {
		pick := rr.Next(choices)
		order[i] = pick.NodeID
	}

	want := []string{"a", "b", "c", "a", "b", "c"}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("round %d = %s, want %s (full order %v)", i, order[i], want[i], order)
		}
	}
}
