package services

import (
	"sync"

	"github.com/ansible/receptor/pkg/netceptor"
)

// ServiceEndpoint identifies a mesh service instance.
type ServiceEndpoint struct {
	NodeID  string
	Service string
}

// TagsMatch reports whether advertisement tags contain every selector key with the same value.
// An empty selector matches all advertisements (subject to optional node/service filters).
func TagsMatch(adTags map[string]string, selector map[string]string) bool {
	for key, want := range selector {
		got, ok := adTags[key]
		if !ok || got != want {
			return false
		}
	}

	return true
}

// DiscoverServicesByTags returns mesh endpoints whose advertisements match selector tags.
// nodeFilter and serviceFilter are optional; when non-empty they restrict candidates.
func DiscoverServicesByTags(
	ads []*netceptor.ServiceAdvertisement,
	selector map[string]string,
	nodeFilter string,
	serviceFilter string,
) []ServiceEndpoint {
	endpoints := make([]ServiceEndpoint, 0)
	seen := make(map[string]struct{})

	for _, ad := range ads {
		if ad == nil {
			continue
		}
		if nodeFilter != "" && ad.NodeID != nodeFilter {
			continue
		}
		if serviceFilter != "" && ad.Service != serviceFilter {
			continue
		}
		if !TagsMatch(ad.Tags, selector) {
			continue
		}
		key := ad.NodeID + "/" + ad.Service
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		endpoints = append(endpoints, ServiceEndpoint{
			NodeID:  ad.NodeID,
			Service: ad.Service,
		})
	}

	return endpoints
}

// RoundRobinPicker selects endpoints in round-robin order.
type RoundRobinPicker struct {
	mu      sync.Mutex
	counter uint64
}

// Next returns the next endpoint from choices using round-robin.
func (rr *RoundRobinPicker) Next(choices []ServiceEndpoint) ServiceEndpoint {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	idx := rr.counter % uint64(len(choices))
	rr.counter++

	return choices[idx]
}
