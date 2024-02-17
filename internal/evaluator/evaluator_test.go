package evaluator

import (
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"tarkov-build-optimiser/internal/db"
	"testing"
)

func TestCalculateOptimumRecoil(t *testing.T) {
	dbClient, err := db.CreateBuildOptimiserDBClient()
	id := "5a0ec13bfcdbcb00165aa685"
	weapon, err := CreateWeaponPossibilityTree(dbClient.Conn, id)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to create possibility tree for weapon id: %s", id)
	}
	err = GenerateOptimumWeaponBuilds(dbClient.Conn, weapon.ID)
	assert.Nil(t, err)
}
