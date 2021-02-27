package metrics

import (
	"github.com/vulcand/oxy/roundrobin"
	"net/url"
)

/**
metrics interface must:
 - upsert
 - remove
 - getMetrics

by itself
 - register pre-post-callbacks
 - call update function on metric value change
 */

type MetricProvider interface {
	RemoveServer(u *url.URL) error
	UpsertServer(u *url.URL, options ...roundrobin.ServerOption) error
	GetWeights() (map[*url.URL]int, error)
}