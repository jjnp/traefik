package lowestrt

import (
	"fmt"
	"github.com/containous/traefik/v2/pkg/config/dynamic"
	"github.com/vulcand/oxy/roundrobin"
	"github.com/vulcand/oxy/utils"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// TODO

type server struct {
	url *url.URL
}

type LrtBalancer struct {
	fwd http.Handler
	cfg *dynamic.LowestResponseTime

	mutex   *sync.Mutex
	servers []*server

	lrtSum map[*server]int
	lrtCounter map[*server]int
	lrtChooser *LRTChooser
	lrtMutex map[*server]*sync.Mutex

	log *log.Logger

	quitLrtCalculation chan struct{}
}

func New(fwd http.Handler, cfg *dynamic.LowestResponseTime) (*LrtBalancer, error) {
	balancer := LrtBalancer{
		fwd: fwd,
		cfg: cfg,

		mutex:   &sync.Mutex{},
		servers: []*server{},

		lrtSum: make(map[*server]int),
		lrtCounter: make(map[*server]int),
		lrtMutex: make(map[*server]*sync.Mutex),
		lrtChooser: NewLRTChooser(cfg.Epsilon),
	}
	balancer.startRegularRecalculation(balancer.cfg.Window)

	return &balancer, nil
}

func (lb *LrtBalancer) NextServer() *server {
	return lb.lrtChooser.Pick()
}

func (lb *LrtBalancer) startRegularRecalculation(interval time.Duration) {
	lb.quitLrtCalculation = make(chan struct{})
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
				case <- ticker.C:
					lb.recalculateWeights()
				case <- lb.quitLrtCalculation:
					ticker.Stop()
					return
			}
		}
	}()
}

func (lb *LrtBalancer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s := lb.NextServer()

	then := time.Now()

	// FIXME: just a prototype impl without sticky sessions

	newReq := *req
	newReq.URL = s.url

	lb.fwd.ServeHTTP(w, &newReq)

	duration := time.Now().Sub(then).Milliseconds()
	// serving by "lowest response time" requires information about the response time, which can only be
	// obtained by sending requests to the server in the first place. so it will be necessary to occasionally
	// send requests to other servers.
	// oxy's Rebalancer seems to generalize this by adapting weights, which may simplify balancing.
	log.Printf("serving on %s took %d ms\n", s.url, duration)
	lb.logRequestDuration(s, int(duration)) // this type conversion should be fine since the millisecond value can't get large enough to cause problems
}

func (lb *LrtBalancer) logRequestDuration(server *server, duration int) {
	lb.lrtMutex[server].Lock()
	defer lb.lrtMutex[server].Unlock()
	lb.lrtSum[server] += duration
	lb.lrtCounter[server]++
}

func (lb *LrtBalancer) Servers() []*url.URL {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	out := make([]*url.URL, len(lb.servers))
	for i, s := range lb.servers {
		out[i] = s.url
	}
	return out
}

func (lb *LrtBalancer) RemoveServer(u *url.URL) error {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	e, index := lb.findServerByURL(u)
	if e == nil {
		return fmt.Errorf("server not found")
	}

	lb.servers = append(lb.servers[:index], lb.servers[index+1:]...)
	lb.lrtMutex[e].Lock()
	delete(lb.lrtCounter, e)
	delete(lb.lrtSum, e)
	lb.lrtChooser.RemoveServer(e)
	lb.lrtMutex[e].Unlock()
	delete(lb.lrtMutex, e)

	return nil
}

// UpsertServer In case if server is already present in the load balancer, returns error
func (lb *LrtBalancer) UpsertServer(u *url.URL, _ ...roundrobin.ServerOption) error {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	if u == nil {
		return fmt.Errorf("server URL can't be nil")
	}

	if s, _ := lb.findServerByURL(u); s != nil {
		return nil
	}

	srv := &server{url: utils.CopyURL(u)}

	lb.servers = append(lb.servers, srv)
	lb.lrtMutex[srv] = &sync.Mutex{}
	lb.lrtMutex[srv].Lock()
	defer lb.lrtMutex[srv].Unlock()
	lb.lrtSum[srv] = 1
	lb.lrtCounter[srv] = 1
	lb.lrtChooser.AddServer(srv)

	return nil
}

func (lb *LrtBalancer) findServerByURL(u *url.URL) (*server, int) {
	return findServerInListByUrl(lb.servers, u)
}

func findServerInListByUrl(list []*server, u *url.URL) (*server, int) {
	if len(list) == 0 {
		return nil, -1
	}
	for i, s := range list {
		if urlEquals(u, s.url) {
			return s, i
		}
	}
	return nil, -1
}

func (lb *LrtBalancer) recalculateWeights() {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	for _, l := range lb.lrtMutex {
		l.Lock()
		defer l.Unlock()
	}

	weights := calculateWeights(lb.lrtSum, lb.lrtCounter)
	lb.resetCalculationValues()
	lb.lrtChooser.PatchWeights(weights)
	//for s, w := range weights {
	//	log.Printf("%s now has weight %d\n", s.url, w)
	//}
}

func (lb *LrtBalancer) resetCalculationValues() {
	for k, _ := range lb.lrtSum {
		lb.lrtSum[k] = 1
		lb.lrtCounter[k] = 1
	}
}

func urlEquals(a, b *url.URL) bool {
	return a.Path == b.Path && a.Host == b.Host && a.Scheme == b.Scheme
}
