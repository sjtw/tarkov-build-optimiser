package helpers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"tarkov-build-optimiser/internal/models"
)

func ContainsStr(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func ContainsSlot(slots []models.Slot, slot models.Slot) bool {
	for _, s := range slots {
		if s.ID == slot.ID {
			return true
		}
	}
	return false
}

func ExtractKeyFromMap(key string, m map[string]interface{}, receiver interface{}) error {
	value, ok := m[key]
	if !ok {
		return fmt.Errorf("key '%s' not found in the map", key)
	}

	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value for key '%s': %w", key, err)
	}

	err = json.Unmarshal(jsonData, &receiver)
	if err != nil {
		return fmt.Errorf("failed to unmarshal value for key '%s': %w", key, err)
	}

	return nil
}

func GetProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	for {
		goModPath := filepath.Join(cwd, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			break
		}

		parent := filepath.Dir(cwd)
		if parent == cwd {
			return "", err
		}
		cwd = parent
	}

	return cwd, nil
}
