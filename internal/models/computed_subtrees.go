package models

import (
	"context"
	"database/sql"
	"encoding/json"
)

type SubtreeAssignment struct {
	SlotID string `json:"slot_id"`
	ItemID string `json:"item_id"`
}

type ComputedSubtree struct {
	RootItemID             string              `json:"root_item_id"`
	BuildType              string              `json:"build_type"`
	JaegerLevel            int                 `json:"jaeger_level"`
	PraporLevel            int                 `json:"prapor_level"`
	PeacekeeperLevel       int                 `json:"peacekeeper_level"`
	MechanicLevel          int                 `json:"mechanic_level"`
	SkierLevel             int                 `json:"skier_level"`
	DepthEvaluated         int                 `json:"depth_evaluated"`
	GameDataVersion        string              `json:"game_data_version"`
	RecoilSum              int                 `json:"recoil_sum"`
	ErgonomicsSum          int                 `json:"ergonomics_sum"`
	ChosenAssignments      []SubtreeAssignment `json:"chosen_assignments"`
	ChosenItemIDs          []string            `json:"chosen_item_ids"`
	ConflictsItemIDs       []string            `json:"conflicts_item_ids"`
	PotentialMinRecoil     int                 `json:"potential_min_recoil"`
	PotentialMaxRecoil     int                 `json:"potential_max_recoil"`
	PotentialMinErgonomics int                 `json:"potential_min_ergonomics"`
	PotentialMaxErgonomics int                 `json:"potential_max_ergonomics"`
}

func UpsertComputedSubtree(db *sql.DB, s *ComputedSubtree) error {
	assignments, err := json.Marshal(s.ChosenAssignments)
	if err != nil {
		return err
	}

	query := `
        insert into computed_subtrees (
            root_item_id, build_type, jaeger_level, prapor_level, peacekeeper_level, mechanic_level, skier_level,
            depth_evaluated, game_data_version, recoil_sum, ergonomics_sum, chosen_assignments, chosen_item_ids,
            conflicts_item_ids, potential_min_recoil, potential_max_recoil, potential_min_ergonomics, potential_max_ergonomics
        ) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
        on conflict (root_item_id, build_type, jaeger_level, prapor_level, peacekeeper_level, mechanic_level, skier_level, depth_evaluated, game_data_version)
        do update set
            recoil_sum = excluded.recoil_sum,
            ergonomics_sum = excluded.ergonomics_sum,
            chosen_assignments = excluded.chosen_assignments,
            chosen_item_ids = excluded.chosen_item_ids,
            conflicts_item_ids = excluded.conflicts_item_ids,
            potential_min_recoil = excluded.potential_min_recoil,
            potential_max_recoil = excluded.potential_max_recoil,
            potential_min_ergonomics = excluded.potential_min_ergonomics,
            potential_max_ergonomics = excluded.potential_max_ergonomics,
            updated_at = now();
    `

	_, err = db.Exec(query,
		s.RootItemID, s.BuildType, s.JaegerLevel, s.PraporLevel, s.PeacekeeperLevel, s.MechanicLevel, s.SkierLevel,
		s.DepthEvaluated, s.GameDataVersion, s.RecoilSum, s.ErgonomicsSum, assignments, pqStringArray(s.ChosenItemIDs),
		pqStringArray(s.ConflictsItemIDs), s.PotentialMinRecoil, s.PotentialMaxRecoil, s.PotentialMinErgonomics, s.PotentialMaxErgonomics,
	)
	if err != nil {
		return err
	}
	return nil
}

// Find the best (highest constraints <= requested) subtree
func GetNearestComputedSubtree(ctx context.Context, db *sql.DB, itemID string, buildType string, levels []TraderLevel, gameDataVersion string, depthCap int) (*ComputedSubtree, error) {
	// Query any subtree for this item/buildType whose levels are <= requested, prefer highest levels within that set, and depth_evaluated >= depthCap.
	// Implemented via a simple filter + order by in SQL.
	query := `
        select root_item_id, build_type,
               jaeger_level, prapor_level, peacekeeper_level, mechanic_level, skier_level,
               depth_evaluated, game_data_version, recoil_sum, ergonomics_sum,
               chosen_assignments, chosen_item_ids, conflicts_item_ids,
               potential_min_recoil, potential_max_recoil, potential_min_ergonomics, potential_max_ergonomics
        from computed_subtrees
        where root_item_id = $1 and build_type = $2 and game_data_version = $3 and depth_evaluated >= $4
          and jaeger_level <= $5 and prapor_level <= $6 and peacekeeper_level <= $7 and mechanic_level <= $8 and skier_level <= $9
        order by jaeger_level desc, prapor_level desc, peacekeeper_level desc, mechanic_level desc, skier_level desc
        limit 1;
    `

	row := db.QueryRowContext(ctx, query, itemID, buildType, gameDataVersion, depthCap,
		levelsByName(levels, "Jaeger"), levelsByName(levels, "Prapor"), levelsByName(levels, "Peacekeeper"), levelsByName(levels, "Mechanic"), levelsByName(levels, "Skier"))

	var s ComputedSubtree
	var assignments string
	var chosenIDs, conflictIDs []byte
	err := row.Scan(&s.RootItemID, &s.BuildType, &s.JaegerLevel, &s.PraporLevel, &s.PeacekeeperLevel, &s.MechanicLevel, &s.SkierLevel,
		&s.DepthEvaluated, &s.GameDataVersion, &s.RecoilSum, &s.ErgonomicsSum, &assignments, &chosenIDs, &conflictIDs,
		&s.PotentialMinRecoil, &s.PotentialMaxRecoil, &s.PotentialMinErgonomics, &s.PotentialMaxErgonomics)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(assignments), &s.ChosenAssignments); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(chosenIDs, &s.ChosenItemIDs); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(conflictIDs, &s.ConflictsItemIDs); err != nil {
		return nil, err
	}
	return &s, nil
}

func levelsByName(levels []TraderLevel, name string) int {
	for _, l := range levels {
		if l.Name == name {
			return l.Level
		}
	}
	return 0
}

// pqStringArray marshals a []string into a JSONB for the INSERT; we store as jsonb, not postgres text[]
func pqStringArray(a []string) []byte {
	b, _ := json.Marshal(a)
	return b
}
