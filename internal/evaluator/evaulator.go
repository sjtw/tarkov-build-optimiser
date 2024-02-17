package evaluator

import (
	"database/sql"
	"encoding/json"
	"github.com/rs/zerolog/log"
)

func findBestRecoilTree(item *Item) (int, *Item) {
	if item.Slots == nil {
		return item.RecoilModifier, item
	}

	if len(item.Slots) == 0 {
		return item.RecoilModifier, item
	}

	sum := item.RecoilModifier

	for i := 0; i < len(item.Slots); i++ {
		item.Slots[i].BestRecoilModifier = 0

		if item.Slots[i].AllowedItems == nil {
			return sum, item
		}

		if len(item.Slots[i].AllowedItems) == 0 {
			return sum, item
		}

		for j := 0; j < len(item.Slots[i].AllowedItems); j++ {
			childSum, highestItem := findBestRecoilTree(item.Slots[i].AllowedItems[j])

			sumWithChildSum := sum + childSum

			if sumWithChildSum < item.Slots[i].BestRecoilModifier {
				item.Slots[i].BestRecoilModifier = childSum
				item.Slots[i].BestRecoilItem = highestItem
			}
		}
	}

	for i := 0; i < len(item.Slots); i++ {
		sum = sum + item.Slots[i].BestRecoilModifier
	}

	return sum, item
}

func findBestErgoTree(item *Item) (int, *Item) {
	sum := item.ErgonomicsModifier

	if item.Slots == nil {
		return sum, item
	}

	if len(item.Slots) == 0 {
		return sum, item
	}

	for i := 0; i < len(item.Slots); i++ {
		item.Slots[i].BestErgoModifier = 0

		if item.Slots[i].AllowedItems == nil {
			break
		}

		if len(item.Slots[i].AllowedItems) == 0 {
			break
		}

		for j := 0; j < len(item.Slots[i].AllowedItems); j++ {
			childSum, highestItem := findBestErgoTree(item.Slots[i].AllowedItems[j])

			sumWithChildSum := sum + childSum

			if sumWithChildSum > item.Slots[i].BestErgoModifier {
				item.Slots[i].BestErgoModifier = childSum
				item.Slots[i].BestErgoItem = highestItem
			}
		}
	}

	for i := 0; i < len(item.Slots); i++ {
		sum = sum + item.Slots[i].BestErgoModifier
	}

	return sum, item
}

func GenerateOptimumWeaponBuilds(db *sql.DB, weaponId string) error {
	weapon, err := CreateWeaponPossibilityTree(db, weaponId)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to create possibility tree for weapon id: %s", weaponId)
		return err
	}

	bestRecoilSum, bestRecoilItem := findBestRecoilTree(weapon)
	err = upsertOptimumBuild(db, weaponId, "recoil", bestRecoilSum, bestRecoilItem, bestRecoilItem.Name)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to upsert optimum recoil build")
		return err
	}

	bestErgoSum, bestErgoItem := findBestErgoTree(weapon)
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
