package custom

import (
	"errors"
	"math"
	"net/url"
	"sync"
)

type WRR struct {
	servers []*url.URL
	weights []int
	mtx sync.Mutex
	last int
	cw int
	max int
	gcd int
	n int
}

func NewWRR(serversByWeight map[*url.URL]int) (*WRR, error) {
	wrr := WRR{
		mtx:     sync.Mutex{},
	}
	//servers := make([]*url.URL, len(serversByWeight))
	servers := []*url.URL{}
	//weights := make([]int, len(serversByWeight))
	weights := []int{}
	for k, v := range serversByWeight {
		servers = append(servers, k)
		weights = append(weights, v) // TODO possible change this to simple assignment for better performance
	}
	max, errmax := max(weights)
	gcd, errgcd := gcd(weights)
	if errmax != nil || errgcd != nil {
		return &wrr, errors.New("error calculating initial values for wrr")
	}
	wrr.max = max
	wrr.gcd = gcd
	wrr.cw = 0
	wrr.last = -1
	wrr.servers = servers
	wrr.weights = weights
	wrr.n = len(servers)

	return &wrr, nil
}

func gcd(ns []int) (int, error) {
	// TODO: Implement this property in order to facilitate faster rotation of servers
	max_possible, err := min(ns)
	if err != nil {
		return -1, err
	}
	gcd := 1
	// We move downward, because this way we can potentially break out of the loop earlier
	// I benchmarked it and it's about twice as fast
	for i := max_possible; i >= 1; i-- {
		valid := true
		for _, n := range ns {
			if n % i != 0 {
				valid = false
				break
			}
		}
		if valid && i > 1{
			gcd = i
			break
		}
	}
	return gcd, nil
}

func min(ns []int) (int, error) {
	if ns == nil {
		return -1, errors.New("cannot calculate min of nil or empty slice")
	}
	min := math.MaxInt64
	for _, cur := range ns {
		if cur < min {
			min = cur
		}
	}
	return min, nil
}

func max(ns []int) (int, error) {
	if ns == nil {
		return -1, errors.New("cannot calculate max of nil or empty slice")
	}
	max := 0
	for _, cur := range ns {
		if cur > max {
			max = cur
		}
	}
	return max, nil
}

func (w *WRR) Next() *url.URL {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	for ;true; {
		w.last = (w.last + 1) % w.n
		if w.last == 0 {
			w.cw -= w.gcd
			if w.cw <= 0 {
				w.cw = w.max
			}
		}
		if w.weights[w.last] >= w.cw {
			return w.servers[w.last]
		}
	}
	panic("reached a theoretically unreachable state trying to calculate the next server in WRR.Next() method")
}
