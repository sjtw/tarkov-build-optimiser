package cache_test

import (
	"github.com/stretchr/testify/assert"
	"tarkov-build-optimiser/internal/cache"
	"tarkov-build-optimiser/internal/models"
	"testing"
)

func TestMemoryCache(t *testing.T) {
	memoryCache := cache.NewMemoryCache()

	memoryCache.Store("weapon1", models.Weapon{
		ID:   "weapon1",
		Name: "M4A1",
	})
	memoryCache.Store("weapon2", models.Weapon{ID: "weapon2", Name: "Glock"})

	weaponA := memoryCache.Get("weapon1").(models.Weapon)
	assert.Equal(t, weaponA.ID, "weapon1")
	assert.Equal(t, weaponA.Name, "M4A1")

	weaponB := memoryCache.Get("weapon2").(models.Weapon)
	assert.Equal(t, weaponB.ID, "weapon2")
	assert.Equal(t, weaponB.Name, "Glock")

	memoryCache.Store("mod1", models.WeaponMod{ID: "mod1", Name: "Magpul MOE Carbine stock"})
	modC := memoryCache.Get("mod1").(models.WeaponMod)

	assert.Equal(t, modC.ID, "mod1")
	assert.Equal(t, modC.Name, "Magpul MOE Carbine stock")
}
