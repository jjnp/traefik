package custom

import (
	"github.com/containous/traefik/v2/pkg/config/dynamic"
	"github.com/containous/traefik/v2/pkg/server/loadbalancer/custom/metrics"
	"github.com/vulcand/oxy/roundrobin"
	"net/http"
	"net/url"
)

/**
Todos:
 - Create struct that implements the go loadbalancer interface
 - Create metrics provider interface
 - Link up to the WRR provider
 - Define the config struct


it must
 - register pre-post callback
 - provide metric update poll trigger
 - handle Req
 - upsert srv
 - remove srv
 */

type CustomBalancer struct {
	metricsProviders []metrics.MetricProvider
	wrr WRR
}

func New(fwd http.Handler, cfg *dynamic.Custom) (*CustomBalancer, error) {
	return &CustomBalancer{}, nil
}

func (c *CustomBalancer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	panic("implement me")
}

func (c *CustomBalancer) Servers() []*url.URL {
	panic("implement me")
}

func (c *CustomBalancer) RemoveServer(u *url.URL) error {
	panic("implement me")
}

func (c *CustomBalancer) UpsertServer(u *url.URL, options ...roundrobin.ServerOption) error {
	panic("implement me")
}

func (c *CustomBalancer) next() *url.URL  {
	panic("implement me")
}

func (c *CustomBalancer) RegisterPreRequestCallback(cb func (req *http.Request)()) error {
	panic ("implement me")
}

func (c *CustomBalancer) RegisterPostRequestCallback(cb func (req *http.Request)()) error {
	panic ("implement me")
}

func (c *CustomBalancer) TriggerWeightUpdate() error {
	panic("implement me")
}
