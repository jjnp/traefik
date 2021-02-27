package custom

import (
	"net/url"
	"sync"
)

type WRR struct {
	weights map[*url.URL]int
	mtx sync.Mutex
	last int
	cw int
	max int
	gcd int
}

func NewWRR(serversByWeight map[*url.URL]int) (WRR, error) {
	panic("Not implemented")
}

func (w *WRR) Next() *url.URL {
	panic("not implemented")
}
