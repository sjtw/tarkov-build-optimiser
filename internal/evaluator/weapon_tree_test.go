package evaluator

import (
	"github.com/stretchr/testify/assert"
	"sort"
	"strings"
	"testing"
)

// Tests the correct candidate sets are generated given a SYMMETRICAL
// conflict map.
func TestGenerateNonConflictingCandidateSets(t *testing.T) {
	candidates := map[string]bool{
		"A": true,
		"B": true,
		"C": true,
		"D": true,
	}
	conflicts := map[string]map[string]bool{
		"A": {"B": true},
		"B": {"A": true, "C": true},
		"C": {"B": true},
	}

	result := GenerateNonConflictingCandidateSets(candidates, conflicts)

	expected := map[string]bool{
		"ACD": false,
		// items with zero conflicts (D in this case) must be present in all candidate sets
		"BD": false,
	}

	for _, r := range result {
		sort.Strings(r)
		key := strings.Join(r, "")
		if _, exists := expected[key]; !exists {
			t.Errorf("Unexpected result: %s", key)
		} else {
			expected[key] = true
		}
	}

	for key, found := range expected {
		assert.Equal(t, true, found, "Expected set %s was not found", key)
	}

	assert.Equal(t, len(expected), len(result), "Unexpected number of result sets")
}

func TestGenerateNonSymmetricalNonConflictingCandidateSets(t *testing.T) {
	candidates := map[string]bool{
		"A": true,
		"B": true,
		"C": true,
		"D": true,
	}
	// C is omitted entirely as its conflicts are inherent
	// this replicates some holes in the tarkov.dev data where a conflict is
	// only represented in one direction. stocks & pistol grips with integrated stock
	// for example.
	conflicts := map[string]map[string]bool{
		"A": {"B": true},
		"C": {"B": true},
	}

	result := GenerateNonConflictingCandidateSets(candidates, conflicts)

	expected := map[string]bool{
		"ACD": false,
		"BD":  false,
	}

	for _, r := range result {
		sort.Strings(r)
		key := strings.Join(r, "")
		if _, exists := expected[key]; !exists {
			t.Errorf("Unexpected result: %s", key)
		} else {
			expected[key] = true
		}
	}

	for key, found := range expected {
		assert.Equal(t, true, found, "Expected set %s was not found", key)
	}

	assert.Equal(t, len(expected), len(result), "Unexpected number of result sets")
}
