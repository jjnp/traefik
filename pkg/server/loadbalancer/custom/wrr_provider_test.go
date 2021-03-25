package custom

import (
	"github.com/stretchr/testify/assert"
	"net/url"
	"strconv"
	"testing"
)

func TestWRR(t *testing.T) {
	var servers []*url.URL
	weights := make(map[*url.URL]int)
	requests := make(map[*url.URL]int)
	for i := 0; i < 7; i++ {
		servers = append(servers, &url.URL{Host: "http://localhost:" + strconv.Itoa(i)})
	}
	for _, s := range servers {
		requests[s] = 0
	}
	weights[servers[0]] = 10
	weights[servers[1]] = 10
	weights[servers[2]] = 8
	weights[servers[3]] = 6
	weights[servers[4]] = 4
	weights[servers[5]] = 2
	weights[servers[6]] = 2

	wrr, err := NewWRR(weights)
	if err != nil {
		assert.Fail(t, "An error occurred trying to create the WRR Provider")
	}
	for i := 0; i < 84; i++ {
		next := wrr.Next()
		requests[next] += 1
	}
	assert.True(t, requests[servers[0]] == requests[servers[6]] * 5)
	assert.True(t, requests[servers[2]] == requests[servers[4]] * 2)
	assert.True(t, requests[servers[3]] == requests[servers[5]] * 3)
}

func TestGCD(t *testing.T) {
	ns1 := []int{40, 12, 8, 36, 24}
	ns2 := []int{40, 12, 8, 36, 7}
	ns3 := []int{40, 8, 8, 32, 24}
	gcd1, _ := gcd(ns1)
	gcd2, _ := gcd(ns2)
	gcd3, _ := gcd(ns3)
	assert.Equal(t, 4, gcd1)
	assert.Equal(t, 1, gcd2)
	assert.Equal(t, 8, gcd3)
}

func BenchmarkGCD(b *testing.B) {
	ns1 := []int{40, 12, 8, 36, 24}
	ns2 := []int{40, 12, 8, 36, 7}
	ns3 := []int{40, 8, 8, 32, 24}
	for i := 0; i < b.N; i++ {
		gcd(ns1)
		gcd(ns2)
		gcd(ns3)
	}
}

func BenchmarkWRR(b *testing.B) {
	var servers []*url.URL
	weights := make(map[*url.URL]int)
	for i := 0; i < 7; i++ {
		servers = append(servers, &url.URL{Host: "http://localhost:" + strconv.Itoa(i)})
	}
	weights[servers[0]] = 10
	weights[servers[1]] = 10
	weights[servers[2]] = 8
	weights[servers[3]] = 6
	weights[servers[4]] = 4
	weights[servers[5]] = 2
	weights[servers[6]] = 2

	wrr, err := NewWRR(weights)
	if err != nil {
		assert.Fail(b, "An error occurred trying to create the WRR Provider")
	}
	for i := 0; i < b.N; i++ {
		wrr.Next()
	}
}
