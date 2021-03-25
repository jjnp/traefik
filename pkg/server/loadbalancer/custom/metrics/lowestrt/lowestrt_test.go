package lowestrt

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"testing"
	"time"
)

type mockedCallbackHelper struct {}

func (c mockedCallbackHelper) RegisterPreRequestCallback(cb func(server *url.URL, req *http.Request)) error {
	return nil
}

func (c mockedCallbackHelper) RegisterPostRequestCallback(cb func(server *url.URL, req *http.Request)) error {
	return nil
}

func TestLowestRTCallbacks(t *testing.T) {
	lrt, err := NewLowestRT(&mockedCallbackHelper{}, 10 * time.Second, 1.0)
	if err != nil {
		assert.Fail(t, "error creating LowestRT instance", err)
	}

	timingsMs := []int{75, 75, 25, 50, 25 }
	s := &url.URL{}
	//lrt.UpsertServer(s)
	assert.Equal(t, 0.0, lrt.rts[s])
	for _, t := range timingsMs {
		r := &http.Request{}
		fmt.Print(t)
		lrt.preRequestHandler(s, r)
		time.Sleep(time.Millisecond * time.Duration(t))
		lrt.postRequestHandler(s, r)
		//time.Sleep(time.Second * 1)
	}
	assert.NotEqual(t, nil, lrt.rts[s])
	assert.NotEqual(t, 0, lrt.rts[s])
}

func TestLowestRTGetWeights(t *testing.T) {
	lrt, err := NewLowestRT(&mockedCallbackHelper{}, 10 * time.Second, 1.0)
	if err != nil {
		assert.Fail(t, "error creating LowestRT instance", err)
	}
	u1, u2, u3 := &url.URL{}, &url.URL{}, &url.URL{}
	lrt.rts = map[*url.URL]float64{u1: 24, u2: 53, u3: 105}
	weights, err := lrt.GetWeights()
	if err != nil {
		assert.Fail(t, "error trying to get weights", err)
	}
	assert.Equal(t, 10, weights[u1])
	assert.Condition(t, func()bool{return weights[u1] == weights[u2] * 2})
	assert.Equal(t, 2, weights[u3])
}

func BenchmarkExpMovingAverage(b *testing.B) {
	val := 32.0
	oldVal := 25.0
	windowMs := 5000.0
	deltaMs := 2500.0
	for i := 0; i < b.N; i++ {
		expMovingAvg(val, oldVal, deltaMs, windowMs)
	}
}
