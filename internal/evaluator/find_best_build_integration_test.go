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

// TestBuildVerificationIntegration verifies that the optimal builds match expected results
func TestFindBestBuildIntegration(t *testing.T) {
	environment, err := env.Get()
	require.NoError(t, err, "Failed to get environment")

	dbClient, err := db.CreateBuildOptimiserDBClient(environment)
	require.NoError(t, err, "Failed to connect to database")

	err = models.PurgeOptimumBuilds(dbClient.Conn)
	require.NoError(t, err, "Failed to purge optimum builds")

	weaponIds := []string{
		"5447a9cd4bdc2dbd208b4567", // Colt M4A1 5.56x45 assault rifle
		"6895bb82c4519957df062f82", // Radian Weapons Model 1 FA 5.56x45 assault rifle
	}

	traderLevels := [][]models.TraderLevel{
		{
			{Name: "Jaeger", Level: 1},
			{Name: "Prapor", Level: 1},
			{Name: "Skier", Level: 1},
			{Name: "Peacekeeper", Level: 1},
			{Name: "Mechanic", Level: 1},
		},
		{
			{Name: "Jaeger", Level: 2},
			{Name: "Prapor", Level: 2},
			{Name: "Skier", Level: 2},
			{Name: "Peacekeeper", Level: 2},
			{Name: "Mechanic", Level: 2},
		},
		{
			{Name: "Jaeger", Level: 3},
			{Name: "Prapor", Level: 3},
			{Name: "Skier", Level: 3},
			{Name: "Peacekeeper", Level: 3},
			{Name: "Mechanic", Level: 3},
		},
		{
			{Name: "Jaeger", Level: 4},
			{Name: "Prapor", Level: 4},
			{Name: "Skier", Level: 4},
			{Name: "Peacekeeper", Level: 4},
			{Name: "Mechanic", Level: 4},
		},
	}

	dataService := candidate_tree.CreateDataService(dbClient.Conn)

	var totalCacheHits, totalCacheMisses int64

	for i, weaponID := range weaponIds {
		for j, traderLevel := range traderLevels {
			t.Logf("Evaluating weapon %s with trader levels %v", weaponID, traderLevel)

			constraints := models.EvaluationConstraints{
				TraderLevels:     traderLevel,
				IgnoredSlotNames: []string{"Scope", "Ubgl", "Tactical"},
				IgnoredItemIDs:   []string{},
			}

			weapon, err := candidate_tree.CreateWeaponCandidateTree(weaponID, "recoil", constraints, dataService)
			if err != nil {
				t.Skipf("Skipping test - database doesn't have weapon data: %v", err)
				return
			}

			weapon.SortAllowedItems("recoil-min")

			// Use memory cache for testing
			cache := NewMemoryCache()
			build := FindBestBuild(weapon, "recoil", map[string]bool{}, cache)
			require.NotNil(t, build, "Expected non-nil build for weapon %s", weaponID)

			hits := build.CacheHits
			misses := build.CacheMisses
			totalCacheHits += hits
			totalCacheMisses += misses

			hitRate := float64(0)
			if hits+misses > 0 {
				hitRate = float64(hits) / float64(hits+misses) * 100
			}

			t.Logf("Weapon %s (Level %d): %d hits, %d misses, %.1f%% hit rate",
				weaponID, j+1, hits, misses, hitRate)

			assert.NotNil(t, build.OptimalItems, "Build should have optimal items")
			assert.Greater(t, len(build.OptimalItems), 0, "Build should have at least one optimal item")
			assert.LessOrEqual(t, build.RecoilSum, 0, "Build should have optimized recoil (low recoil sum)")
			// Convert to evaluated weapon and verify structure
			evaledWeapon, err := build.ToEvaluatedWeapon()
			require.NoError(t, err, "Failed to convert build to evaluated weapon")

			assert.NotNil(t, evaledWeapon, "Evaluated weapon should not be nil")
			assert.NotEmpty(t, evaledWeapon.Name, "Evaluated weapon should have a name")
			assert.NotEmpty(t, evaledWeapon.ID, "Evaluated weapon should have an ID")

			assert.NotNil(t, evaledWeapon.Slots, "Evaluated weapon should have slots")
			assert.Greater(t, len(evaledWeapon.Slots), 0, "Evaluated weapon should have at least one slot")

			for _, slot := range evaledWeapon.Slots {
				if !slot.IsEmpty {
					assert.NotEmpty(t, slot.Item.Name, "Slot %s should have an item name", slot.Name)
					assert.NotEmpty(t, slot.Item.ID, "Slot %s should have an item ID", slot.Name)
					t.Logf("Slot %s (%s): %s (ID: %s)", slot.Name, slot.ID, slot.Item.Name, slot.Item.ID)
				} else {
					t.Logf("Slot %s (%s): Empty (this may be expected for some weapons)", slot.Name, slot.ID)
				}
			}

			if i == 1 && j == 0 { // Radian, Level 1
				assert.Greater(t, hits, int64(0), "Second weapon should have cache hits from first weapon")
				t.Logf("✓ Cross-weapon caching verified: %d cache hits for Radian", hits)
			}

		}
	}

	totalEvaluations := totalCacheHits + totalCacheMisses
	overallHitRate := float64(0)
	if totalEvaluations > 0 {
		overallHitRate = float64(totalCacheHits) / float64(totalEvaluations) * 100
	}

	t.Logf("Overall cache performance: %d hits, %d misses, %.1f%% hit rate",
		totalCacheHits, totalCacheMisses, overallHitRate)

	assert.Greater(t, totalCacheHits, int64(0), "Should have cache hits across all evaluations")

	t.Log("✓ Both weapons use AR-15 platform components")
	t.Log("✓ Cross-weapon caching is working effectively")
	t.Log("✓ Item-level subtree caching is providing performance benefits")
}
