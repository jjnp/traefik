package lowestrt

const baseWeight int = 100000

func calculateWeights(values map[*server]int, counters map[*server]int) map[*server]int {
	results := make(map[*server]int, len(values))
	for k, v := range values {
		results[k] = baseWeight / (v / counters[k])
	}
	return results
}
