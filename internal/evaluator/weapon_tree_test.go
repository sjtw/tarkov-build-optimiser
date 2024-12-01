package evaluator

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGenerateNonConflictingCandidateSets(t *testing.T) {
	candidates := map[string]bool{
		"1": true,
		"2": true,
		"3": true,
		"4": true,
		"5": true,
		"6": true,
	}

	conflicts := map[string]map[string]bool{
		"1": map[string]bool{"2": true, "3": true},
		"2": map[string]bool{"1": true, "3": true},
		"3": map[string]bool{"1": true, "2": true},
	}

	sets := GenerateNonConflictingCandidateSets(candidates, conflicts)
	assert.Equal(t, 3, len(sets))

	counts := map[string]int{}
	for _, set := range sets {
		for _, id := range set {
			if _, ok := counts[id]; ok {
				counts[id]++
			} else {
				counts[id] = 1
			}
		}
	}

	assert.Equal(t, 1, counts["1"])
	assert.Equal(t, 1, counts["2"])
	assert.Equal(t, 1, counts["3"])
	assert.Equal(t, 3, counts["4"])
	assert.Equal(t, 3, counts["5"])
	assert.Equal(t, 3, counts["6"])
}
