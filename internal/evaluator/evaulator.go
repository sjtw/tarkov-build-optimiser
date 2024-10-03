package evaluator

import (
	"database/sql"
	"encoding/json"
	"github.com/rs/zerolog/log"
	"tarkov-build-optimiser/internal/models"
)

func findBestRecoilTree(item *Item) *Item {
	if item.Slots == nil || len(item.Slots) == 0 {
		return item
	}

	for i := 0; i < len(item.Slots); i++ {
		item.Slots[i].BestRecoilModifier = 0

		if item.Slots[i].AllowedItems == nil || len(item.Slots[i].AllowedItems) == 0 {
			return item
		}

		for j := 0; j < len(item.Slots[i].AllowedItems); j++ {
			highestItem := findBestRecoilTree(item.Slots[i].AllowedItems[j])
			childSum := highestItem.GetBestRecoilSum()

			if childSum < item.Slots[i].BestRecoilModifier {
				item.Slots[i].BestRecoilModifier = childSum
				item.Slots[i].BestRecoilItem = highestItem
			}
		}
	}

	return item
}

func findBestErgoTree(item *Item) *Item {

	if item.Slots == nil || len(item.Slots) == 0 {
		return item
	}

	for i := 0; i < len(item.Slots); i++ {
		item.Slots[i].BestErgoModifier = 0

		if item.Slots[i].AllowedItems == nil || len(item.Slots[i].AllowedItems) == 0 {
			return item
		}

		for j := 0; j < len(item.Slots[i].AllowedItems); j++ {
			highestItem := findBestErgoTree(item.Slots[i].AllowedItems[j])
			childSum := highestItem.GetBestErgoSum()

			if childSum > item.Slots[i].BestErgoModifier {
				item.Slots[i].BestErgoModifier = childSum
				item.Slots[i].BestErgoItem = highestItem
			}
		}

	}

	return item
}

func GenerateOptimumWeaponBuilds(db *sql.DB, weaponId string) error {
	weapon, err := CreateWeaponPossibilityTree(db, weaponId)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to create possibility tree for weapon id: %s", weaponId)
		return err
	}

	bestRecoilItem := findBestRecoilTree(weapon)
	bestRecoilSum := bestRecoilItem.GetBestRecoilSum()
	err = upsertOptimumBuild(db, weaponId, "recoil", bestRecoilSum, bestRecoilItem, bestRecoilItem.Name)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to upsert optimum recoil build")
		return err
	}

	bestErgoItem := findBestErgoTree(weapon)
	bestErgoSum := bestErgoItem.GetBestErgoSum()
	err = upsertOptimumBuild(db, weaponId, "ergo", bestErgoSum, bestErgoItem, bestErgoItem.Name)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to upsert optimum ergo build")
		return err
	}

	return nil
}

func upsertOptimumBuild(db *sql.DB, itemId string, buildType string, sum int, build *Item, name string) error {
	b, err := json.Marshal(build)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal build")
		return err
	}

	query := `INSERT INTO optimum_builds (
			item_id,
			build,
			build_type,
            modifier_sum,
            name
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (item_id, build_type) DO UPDATE SET
			build = $2,
			modifier_sum = $4,
			name = $5
		;`
	_, err = db.Exec(query, itemId, b, buildType, sum, name)
	if err != nil {
		return err
	}

	return nil
}

func CreateWeaponPossibilityTree(db *sql.DB, id string) (*Item, error) {
	w, err := models.GetWeaponById(db, id)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get weapon %s", id)
		return nil, err
	}
	weapon := &Item{
		ID:                 id,
		Name:               w.Name,
		RecoilModifier:     w.RecoilModifier,
		ErgonomicsModifier: w.ErgonomicsModifier,
		Slots:              []*ItemSlot{},
		parentSlot:         nil,
		Type:               "weapon",
	}

	err = weapon.PopulateSlots(db)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to populate slots for weapon %s", w.ID)
		return nil, err
	}

	return weapon, nil
}
