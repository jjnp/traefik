package lowestrt

import (
	"github.com/containous/traefik/v2/pkg/server/loadbalancer/custom/metrics"
	"github.com/vulcand/oxy/roundrobin"
	"math"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type LowestRT struct {
	rts map[*url.URL]float64
	lastReq map[*url.URL]time.Time
	timingBuffer map[timingKey]time.Time // Todo: Run a periodic cleanup for failed requests s.t. the buffer gets cleaned -> prevent memory leak
	window time.Duration
	scaling float64
	mtx sync.Mutex
	timingMtx sync.Mutex
}

type timingKey struct {
	s *url.URL
	r *http.Request
}

func NewLowestRT(c metrics.CallbackRegistrationHelper, window time.Duration, scaling float64) (*LowestRT, error) {
	lrt := LowestRT{}
	lrt.window = window
	lrt.scaling = scaling
	lrt.rts = make(map[*url.URL]float64)
	lrt.lastReq = make(map[*url.URL]time.Time)
	lrt.timingBuffer = make(map[timingKey]time.Time)
	c.RegisterPreRequestCallback(lrt.preRequestHandler)
	c.RegisterPostRequestCallback(lrt.postRequestHandler)
	return &lrt, nil
}

func expMovingAvg(value, oldValue, deltaMs, windowMs float64) float64 {
	alpha := 1.0 - math.Exp(-deltaMs/windowMs)
	r := alpha * value + (1.0 - alpha) * oldValue
	return r
}

func (l *LowestRT) preRequestHandler(server *url.URL, req *http.Request)  {
	l.timingMtx.Lock()
	defer l.timingMtx.Unlock()
	l.timingBuffer[timingKey{
		s: server,
		r: req,
	}] = time.Now()
}

func (l *LowestRT) postRequestHandler(server *url.URL, req *http.Request)  {
	l.timingMtx.Lock()
	defer l.timingMtx.Unlock()
	t2 := time.Now()
	key := timingKey{
		s: server,
		r: req,
	}
	t1, ok := l.timingBuffer[key]
	if !ok {
		return
	}
	rtt := float64(t2.Sub(t1).Milliseconds())
	l.mtx.Lock()
	defer l.mtx.Unlock()
	var deltaMs float64 = 10000000
	if val, ok := l.lastReq[server]; ok {
		deltaMs = float64(t2.Sub(val).Milliseconds())
	}
	l.rts[server] = expMovingAvg(rtt, l.rts[server], deltaMs, float64(l.window.Milliseconds()))
	l.lastReq[server] = time.Now()
	delete(l.timingBuffer, key)
}

func (l *LowestRT) RemoveServer(u *url.URL) error {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	delete(l.rts, u)
	delete(l.lastReq, u)
	for k, _ := range l.timingBuffer {
		if k.s == u {
			delete(l.timingBuffer, k)
		}
	}
	return nil
}

func (l *LowestRT) UpsertServer(u *url.URL, options ...roundrobin.ServerOption) error {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	if _, ok := l.rts[u]; !ok {
		// We assume a low initial response time of 15ms to make sure we get a few requests for evaluation
		l.rts[u] = 15
	}
	return nil
}

func (l LowestRT) GetWeights() (map[*url.URL]int, error) {
	weights := make(map[*url.URL]int)
	min := 10000000000.0
	for _, v := range l.rts {
		if v < min {
			min = v
		}
	}
	for k, v := range l.rts {
		w := int(math.Round(math.Max(1.0, math.Pow(10 / (v / min), l.scaling))))
		weights[k] = w
	}
	return weights, nil
}
