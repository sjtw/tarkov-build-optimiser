package evaluator

import (
	"testing"

	"tarkov-build-optimiser/internal/candidate_tree"
	"tarkov-build-optimiser/internal/db"
	"tarkov-build-optimiser/internal/env"
	"tarkov-build-optimiser/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCacheCorrectnessIntegration verifies that cached builds match non-cached builds
func TestCacheCorrectnessIntegration(t *testing.T) {
	environment, err := env.Get()
	require.NoError(t, err, "Failed to get environment")

	dbClient, err := db.CreateBuildOptimiserDBClient(environment)
	require.NoError(t, err, "Failed to connect to database")

	weaponID := "5447a9cd4bdc2dbd208b4567" // Colt M4A1

	// Test at Level 2 - enough items to be interesting
	traderLevel := []models.TraderLevel{
		{Name: "Jaeger", Level: 2},
		{Name: "Prapor", Level: 2},
		{Name: "Skier", Level: 2},
		{Name: "Peacekeeper", Level: 2},
		{Name: "Mechanic", Level: 2},
	}

	dataService := candidate_tree.CreateDataService(dbClient.Conn)

	constraints := models.EvaluationConstraints{
		TraderLevels:     traderLevel,
		IgnoredSlotNames: []string{"Scope", "Ubgl", "Tactical"},
		IgnoredItemIDs:   []string{},
	}

	// Create weapon tree
	weapon, err := candidate_tree.CreateWeaponCandidateTree(weaponID, "recoil", constraints, dataService)
	if err != nil {
		t.Skipf("Skipping test - database doesn't have weapon data: %v", err)
		return
	}
	weapon.SortAllowedItems("recoil-min")

	// Run WITHOUT cache (pass nil)
	t.Log("Running evaluation WITHOUT cache...")
	buildNoCache := FindBestBuild(weapon, "recoil", map[string]bool{}, nil)
	require.NotNil(t, buildNoCache, "Expected non-nil build without cache")
	t.Logf("No cache: RecoilSum=%d, ErgonomicsSum=%d, Items=%d",
		buildNoCache.RecoilSum, buildNoCache.ErgonomicsSum, len(buildNoCache.OptimalItems))

	// Run WITH cache
	t.Log("Running evaluation WITH cache...")
	cache := NewMemoryCache()
	buildWithCache := FindBestBuild(weapon, "recoil", map[string]bool{}, cache)
	require.NotNil(t, buildWithCache, "Expected non-nil build with cache")
	t.Logf("With cache: RecoilSum=%d, ErgonomicsSum=%d, Items=%d, Hits=%d, Misses=%d",
		buildWithCache.RecoilSum, buildWithCache.ErgonomicsSum, len(buildWithCache.OptimalItems),
		buildWithCache.CacheHits, buildWithCache.CacheMisses)

	// Compare builds - CRITICAL CORRECTNESS CHECK
	assert.Equal(t, buildNoCache.RecoilSum, buildWithCache.RecoilSum,
		"RecoilSum should be identical with and without cache")
	assert.Equal(t, buildNoCache.ErgonomicsSum, buildWithCache.ErgonomicsSum,
		"ErgonomicsSum should be identical with and without cache")
	assert.Equal(t, len(buildNoCache.OptimalItems), len(buildWithCache.OptimalItems),
		"Number of optimal items should be identical")

	// Log any differences in items (for debugging)
	noCacheItems := make(map[string]string)
	for _, item := range buildNoCache.OptimalItems {
		noCacheItems[item.SlotID] = item.ID
	}

	withCacheItems := make(map[string]string)
	for _, item := range buildWithCache.OptimalItems {
		withCacheItems[item.SlotID] = item.ID
	}

	for slotID, noCacheItemID := range noCacheItems {
		withCacheItemID := withCacheItems[slotID]
		if noCacheItemID != withCacheItemID {
			t.Errorf("MISMATCH at slot %s: no-cache=%s, with-cache=%s", slotID, noCacheItemID, withCacheItemID)
		}
	}

	t.Log("✓ Cached and non-cached builds are IDENTICAL")
}
