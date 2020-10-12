package lowestrt

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
)

// This is an adaptation of the weighted random picker found here https://github.com/mroth/weightedrand/blob/master/weightedrand.go

type LRTChooser struct {
	choices []choice
	totals []int
	max int
	mutex *sync.Mutex
	epsilon float64
}

type choice struct {
	server *server
	weight int
}

func NewLRTChooser(epsilon float64) *LRTChooser {
	if epsilon > 1 || epsilon <= 0 {
		panic("Fatal error! Epsilon value must be between 0 and 1")
	}
	chooser := LRTChooser{
		choices: []choice{},
		mutex: &sync.Mutex{},
		max: 1,
		epsilon: epsilon,
	}
	return &chooser
}

func (chooser *LRTChooser) AddServer(server *server) {
	chooser.mutex.Lock()
	defer chooser.mutex.Unlock()
	if c, _ := chooser.findChoiceByServer(server); c != nil {
		return
	}
	choiceCount := len(chooser.choices)
	// Init it to at least 1 to avoid a division by 0 error on startup
	if choiceCount == 0 {
		choiceCount = 1
	}
	chooser.choices = append(chooser.choices, choice{
		server: server,
		weight: chooser.max / choiceCount,
	})
	chooser.calculateTotals()
}

func (chooser *LRTChooser) PatchWeights(weights map[*server]int) {
	chooser.mutex.Lock()
	defer chooser.mutex.Unlock()
	for i, c := range chooser.choices {
		if w, ok := weights[c.server]; ok {
			chooser.choices[i].weight = int((float64(c.weight) * chooser.epsilon) + (float64(w) * (1 - chooser.epsilon)))
			log.Printf("Updated weight for server: %s from %d to %d\n", c.server.url.String(), c.weight, chooser.choices[i].weight)
		} else {
			panic(fmt.Errorf("Fatal error! Weight for server %s is missing!\n", c.server.url.String()))
		}
	}
	chooser.calculateTotals()
}

func (chooser *LRTChooser) RemoveServer(server *server) error {
	chooser.mutex.Lock()
	defer chooser.mutex.Unlock()
	c, i := chooser.findChoiceByServer(server)
	if c == nil {
		return fmt.Errorf("cannot remove server with url %s from chooser, since it wasn't found\n", server.url.String())
	}
	chooser.choices = append(chooser.choices[:i], chooser.choices[i+1:]...)
	chooser.calculateTotals()
	return nil
}

func (chooser *LRTChooser) findChoiceByServer(server *server) (*choice, int) {
	if len(chooser.choices) == 0 {
		return nil, -1
	}
	for i, c := range chooser.choices {
		if urlEquals(server.url, c.server.url) {
			return &c, i
		}
	}
	return nil, -1
}

func (chooser *LRTChooser) calculateTotals() {
	totals := make([]int, len(chooser.choices))
	runningTotal := 0
	for index, choice := range chooser.choices {
		runningTotal += choice.weight
		totals[index] = runningTotal
	}
	chooser.totals = totals
	chooser.max = runningTotal
}

func (chooser *LRTChooser) Pick() *server {
	r := rand.Intn(chooser.max) + 1
	i := searchInts(chooser.totals, r)
	return chooser.choices[i].server
}

func searchInts(a []int, x int) int {
	// Possible further future optimization for searchInts via SIMD if we want
	// to write some Go assembly code: http://0x80.pl/articles/simd-search.html
	i, j := 0, len(a)
	for i < j {
		h := int(uint(i+j) >> 1) // avoid overflow when computing h
		if a[h] < x {
			i = h + 1
		} else {
			j = h
		}
	}
	return i
}