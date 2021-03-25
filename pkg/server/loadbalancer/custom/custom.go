package custom

import (
	"context"
	"github.com/containous/traefik/v2/pkg/config/dynamic"
	"github.com/containous/traefik/v2/pkg/log"
	"github.com/containous/traefik/v2/pkg/server/loadbalancer/custom/metrics"
	"github.com/containous/traefik/v2/pkg/server/loadbalancer/custom/metrics/lowestrt"
	"github.com/vulcand/oxy/roundrobin"
	"net/http"
	"net/url"
	"sync"
	"time"
)

/**
Todos:
 - Create struct that implements the go loadbalancer interface DONE
 - Create metrics provider interface DONE
 - Link up to the WRR provider DONE
 - Define the config struct


it must
 - register pre-post callback DONE
 - provide metric update poll trigger DONE
 - handle Req DONE
 - upsert srv DONE
 - remove srv DONE
 */

type CustomBalancer struct {
	servers []*url.URL
	serversMutex sync.Mutex
	metricsProviders []metrics.MetricProvider
	wrr WRR
	fwd http.Handler
	preReqCBs []func (server *url.URL, req *http.Request)()
	postReqCBs []func (server *url.URL, req *http.Request)()
	lastUpdate time.Time
	updateFrequency time.Duration
	log log.Logger
}

func New(fwd http.Handler, cfg *dynamic.Custom, ctx context.Context) (*CustomBalancer, error) {

	c := &CustomBalancer{
		servers:          []*url.URL{},
		serversMutex:     sync.Mutex{},
		metricsProviders: []metrics.MetricProvider{},
		fwd:              fwd,
		preReqCBs:        []func (server *url.URL, req *http.Request)(){},
		postReqCBs:       []func (server *url.URL, req *http.Request)(){},
		lastUpdate: time.Now(),
		updateFrequency: time.Duration(cfg.UpdateFrequencySeconds) * time.Second,
		log: log.FromContext(ctx),
	}
	wrr, err := NewWRR(make(map[*url.URL]int))
	if err != nil {
		return nil, err
	}
	c.wrr = *wrr
	if cfg.LowestResponseTime != nil {
		lrt, err := lowestrt.NewLowestRT(c, time.Duration(cfg.LowestResponseTime.Window) * time.Second, cfg.LowestResponseTime.Scaling)
		if err != nil {
			return nil, err
		}
		c.metricsProviders = append(c.metricsProviders, lrt)
	}
	return c, nil
}

func (c *CustomBalancer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s := c.wrr.Next()
	newReq := *req
	newReq.URL = s
	for _, cb := range c.preReqCBs {
		cb(s, req)
	}
	c.fwd.ServeHTTP(w, &newReq)
	for _, cb := range c.postReqCBs {
		cb(s, req)
	}
	if c.shouldUpdateWeights() {
		c.updateWeights()
	}
}

func (c *CustomBalancer) shouldUpdateWeights() bool {
	//now := time.Now()
	//delta := now.Sub(c.lastUpdate)
	//return delta >= c.updateFrequency
	return time.Now().Sub(c.lastUpdate) >= c.updateFrequency
}

func (c CustomBalancer) Servers() []*url.URL {
	return c.servers
}

func (c *CustomBalancer) RemoveServer(u *url.URL) error {
	c.serversMutex.Lock()
	defer c.serversMutex.Unlock()
	for i, s := range c.servers {
		if s == u {
			c.servers = append(c.servers[:i], c.servers[i+1:]...)
			break
		}
	}
	for _, p := range c.metricsProviders {
		p.RemoveServer(u)
	}
	c.updateWeights()
	return nil
}

func (c *CustomBalancer) UpsertServer(u *url.URL, options ...roundrobin.ServerOption) error {
	c.serversMutex.Lock()
	defer c.serversMutex.Unlock()
	for _, s := range c.servers {
		if s == u {
			return nil
		}
	}
	c.servers = append(c.servers, u)
	for _, p := range c.metricsProviders {
		p.UpsertServer(u)
	}
	c.updateWeights()
	return nil
}

func (c *CustomBalancer) next() *url.URL  {
	panic("implement me")
}

func (c *CustomBalancer) RegisterPreRequestCallback(cb func (server *url.URL, req *http.Request)()) error {
	c.preReqCBs = append(c.preReqCBs, cb)
	return nil
}

func (c *CustomBalancer) RegisterPostRequestCallback(cb func (server *url.URL, req *http.Request)()) error {
	c.postReqCBs = append(c.postReqCBs, cb)
	return nil
}

func (c *CustomBalancer) TriggerWeightUpdate() {
	c.updateWeights()
}

func (c *CustomBalancer) updateWeights() {
	log.Info("updating weights for wrr")
	weightSum := make(map[*url.URL]int)
	for _, p := range c.metricsProviders {
		if weights, err := p.GetWeights(); err == nil {
			for s, w := range weights {
				weightSum[s] += w
			}
		}
	}
	log.Info(weightSum)
	wrr, err := NewWRR(weightSum)
	if err == nil {
		c.wrr = *wrr
	}
	c.lastUpdate = time.Now()
}
