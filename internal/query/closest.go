package query

import (
	"slices"
	"strings"
)

const closestNameCount = 3

func closestNames(name string, candidates []string) []string {
	type candidate struct {
		name     string
		distance int
	}

	lower := strings.ToLower(name)
	limit := max(2, len(name)/3)
	var near []candidate
	seen := map[string]bool{}
	for _, c := range candidates {
		if seen[c] {
			continue
		}
		seen[c] = true
		d := editDistance(lower, strings.ToLower(c))
		if d <= limit || strings.Contains(strings.ToLower(c), lower) {
			near = append(near, candidate{name: c, distance: d})
		}
	}

	slices.SortStableFunc(near, func(a, b candidate) int {
		if a.distance != b.distance {
			return a.distance - b.distance
		}
		return strings.Compare(a.name, b.name)
	})

	names := make([]string, 0, closestNameCount)
	for _, c := range near[:min(len(near), closestNameCount)] {
		names = append(names, c.name)
	}
	return names
}

func editDistance(a, b string) int {
	ar, br := []rune(a), []rune(b)
	previous := make([]int, len(br)+1)
	current := make([]int, len(br)+1)
	for j := range previous {
		previous[j] = j
	}
	for i := 1; i <= len(ar); i++ {
		current[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			current[j] = min(previous[j]+1, current[j-1]+1, previous[j-1]+cost)
		}
		previous, current = current, previous
	}
	return previous[len(br)]
}
