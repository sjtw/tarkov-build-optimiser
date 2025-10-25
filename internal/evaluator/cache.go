package evaluator

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"tarkov-build-optimiser/internal/models"
)

// CacheEntry represents a cached conflict-free item result
type CacheEntry struct {
	RecoilSum     int `json:"recoil_sum"`
	ErgonomicsSum int `json:"ergonomics_sum"`
}

// Cache interface for conflict-free item caching
type Cache interface {
	// Get retrieves a cache entry for the given parameters
	Get(ctx context.Context, itemID string, focusedStat string, constraints models.EvaluationConstraints) (*CacheEntry, error)

	// Set stores a cache entry for the given parameters
	Set(ctx context.Context, itemID string, focusedStat string, constraints models.EvaluationConstraints, entry *CacheEntry) error

	// Clear removes all entries from the cache
	Clear(ctx context.Context) error
}

// MemoryCache implements Cache using in-memory sync.Map
type MemoryCache struct {
	cache *sync.Map
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		cache: &sync.Map{},
	}
}

// Get retrieves from memory cache
func (m *MemoryCache) Get(ctx context.Context, itemID string, focusedStat string, constraints models.EvaluationConstraints) (*CacheEntry, error) {
	key := makeCacheKey(itemID, focusedStat, constraints)
	if val, ok := m.cache.Load(key); ok {
		return val.(*CacheEntry), nil
	}
	return nil, nil // Cache miss
}

// Set stores in memory cache
func (m *MemoryCache) Set(ctx context.Context, itemID string, focusedStat string, constraints models.EvaluationConstraints, entry *CacheEntry) error {
	key := makeCacheKey(itemID, focusedStat, constraints)
	m.cache.Store(key, entry)
	return nil
}

// Clear removes all entries from memory cache
func (m *MemoryCache) Clear(ctx context.Context) error {
	m.cache = &sync.Map{}
	return nil
}

// DatabaseCache implements Cache using database persistence
type DatabaseCache struct {
	db *sql.DB
}

// NewDatabaseCache creates a new database-backed cache
func NewDatabaseCache(db *sql.DB) *DatabaseCache {
	return &DatabaseCache{db: db}
}

// Get retrieves from database cache
func (d *DatabaseCache) Get(ctx context.Context, itemID string, focusedStat string, constraints models.EvaluationConstraints) (*CacheEntry, error) {
	cachedEntry, err := models.GetConflictFreeCache(ctx, d.db, itemID, focusedStat, constraints.TraderLevels)
	if err != nil {
		// Handle domain-specific cache miss (no rows found) vs actual database error
		if err == sql.ErrNoRows {
			return nil, nil // Cache miss - no error
		}
		return nil, err // Actual database error
	}
	if cachedEntry == nil {
		return nil, nil // Cache miss
	}

	return &CacheEntry{
		RecoilSum:     cachedEntry.RecoilSum,
		ErgonomicsSum: cachedEntry.ErgonomicsSum,
	}, nil
}

// Set stores in database cache
func (d *DatabaseCache) Set(ctx context.Context, itemID string, focusedStat string, constraints models.EvaluationConstraints, entry *CacheEntry) error {
	cacheEntry := &models.ConflictFreeCache{
		ItemID:           itemID,
		FocusedStat:      focusedStat,
		JaegerLevel:      getTraderLevel(constraints.TraderLevels, "Jaeger"),
		PraporLevel:      getTraderLevel(constraints.TraderLevels, "Prapor"),
		PeacekeeperLevel: getTraderLevel(constraints.TraderLevels, "Peacekeeper"),
		MechanicLevel:    getTraderLevel(constraints.TraderLevels, "Mechanic"),
		SkierLevel:       getTraderLevel(constraints.TraderLevels, "Skier"),
		RecoilSum:        entry.RecoilSum,
		ErgonomicsSum:    entry.ErgonomicsSum,
	}
	return models.UpsertConflictFreeCache(d.db, cacheEntry)
}

// Clear removes all entries from database cache
func (d *DatabaseCache) Clear(ctx context.Context) error {
	return models.PurgeConflictFreeCache(d.db)
}

// makeCacheKey creates a cache key for the given parameters
func makeCacheKey(itemID string, focusedStat string, constraints models.EvaluationConstraints) string {
	constraintsSig := serializeConstraints(constraints)
	return fmt.Sprintf("cf|%s|%s|%s", itemID, focusedStat, constraintsSig)
}

// serializeConstraints creates a stable string representation of constraints
func serializeConstraints(constraints models.EvaluationConstraints) string {
	// Create a deterministic string from trader levels
	levels := make([]string, len(constraints.TraderLevels))
	for i, level := range constraints.TraderLevels {
		levels[i] = fmt.Sprintf("%s:%d", level.Name, level.Level)
	}
	return fmt.Sprintf("%v", levels)
}
