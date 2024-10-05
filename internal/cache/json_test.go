package cache_test

import (
	"tarkov-build-optimiser/internal/cache"
	"tarkov-build-optimiser/internal/models"
	"testing"
)

func getCache() (cache.FileCache, error) {
	c, err := cache.NewJSONFileCache("./test.json")
	if err != nil {
		return nil, err
	}
	err = c.Purge()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func TestStoreWeapons(t *testing.T) {
	c, err := getCache()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = c.Store("weapon1", models.Weapon{ID: "weapon1", Name: "M4A1"})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	err = c.Store("weapon2", models.Weapon{ID: "weapon2", Name: "Glock"})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	weaponA := models.Weapon{}
	err = c.Get("weapon1", &weaponA)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if weaponA.ID != "weapon1" {
		t.Errorf("Expected weapon ID to be 'weapon1', got %s", weaponA.ID)
	}
	if weaponA.Name != "M4A1" {
		t.Errorf("Expected weapon name to be 'M4A1', got %s", weaponA.Name)
	}

	weaponB := models.Weapon{}
	err = c.Get("weapon2", &weaponB)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if weaponB.ID != "weapon2" {
		t.Errorf("Expected weapon ID to be 'weapon2', got %s", weaponB.ID)
	}
	if weaponB.Name != "Glock" {
		t.Errorf("Expected weapon name to be 'Glock', got %s", weaponB.Name)
	}

}

func TestStoreWeaponMod(t *testing.T) {
	c, err := getCache()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = c.Store("mod1", models.WeaponMod{ID: "mod1", Name: "Magpul MOE Carbine stock"})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	item := models.WeaponMod{}
	err = c.Get("mod1", &item)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if item.ID != "mod1" {
		t.Errorf("Expected weapon ID to be 'mod1', got %s", item.ID)
	}
	if item.Name != "Magpul MOE Carbine stock" {
		t.Errorf("Expected weapon name to be 'M4A1', got %s", item.Name)
	}
}

func TestGetAll(t *testing.T) {
	c, err := getCache()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = c.Store("weapon1", models.Weapon{ID: "weapon1", Name: "M4A1"})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	err = c.Store("weapon2", models.Weapon{ID: "weapon2", Name: "Glock"})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	result, err := c.All()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	length := len(result.Keys())
	if length != 2 {
		t.Errorf("Expected 2 models, got %d", length)
	}

	weaponA := models.Weapon{}
	err = result.Get("weapon1", &weaponA)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if weaponA.ID != "weapon1" {
		t.Errorf("Expected weapon ID to be 'weapon1', got %s", weaponA.ID)
	}
	if weaponA.Name != "M4A1" {
		t.Errorf("Expected weapon name to be 'M4A1', got %s", weaponA.Name)
	}

	weaponB := models.Weapon{}
	err = result.Get("weapon2", &weaponB)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if weaponB.ID != "weapon2" {
		t.Errorf("Expected weapon ID to be 'weapon2', got %s", weaponB.ID)
	}
	if weaponB.Name != "Glock" {
		t.Errorf("Expected weapon name to be 'Glock', got %s", weaponB.Name)
	}
}
