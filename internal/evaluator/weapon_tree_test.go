package evaluator

import (
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"tarkov-build-optimiser/internal/db"
	"testing"
)

// TODO: use mock data - currently uses db
func TestCreateWeaponPossibilityTree(t *testing.T) {
	// AKMN
	dbClient, err := db.CreateBuildOptimiserDBClient()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to DB")
		return
	}

	id := "5a0ec13bfcdbcb00165aa685"
	weapon, err := CreateWeaponPossibilityTree(dbClient.Conn, id)
	assert.NoError(t, err)

	assert.IsType(t, &Item{}, weapon)
}
