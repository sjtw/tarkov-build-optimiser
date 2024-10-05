package cache

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"os"
	"sync"
	"tarkov-build-optimiser/internal/helpers"
)

type JSONFileCache struct {
	filePath string
	mu       sync.Mutex
}

type JSONFileCacheAllResult struct {
	Items map[string]interface{}
}

func createFileCacheAllResult(items map[string]interface{}) FileCacheAllResult {
	return &JSONFileCacheAllResult{
		Items: items,
	}
}

func (a *JSONFileCacheAllResult) Get(key string, receiver interface{}) error {
	return helpers.ExtractKeyFromMap(key, a.Items, &receiver)
}

func (a *JSONFileCacheAllResult) Length() int {
	return len(a.Items)
}

func (a *JSONFileCacheAllResult) Keys() []string {
	keys := make([]string, 0, len(a.Items))
	for k := range a.Items {
		keys = append(keys, k)
	}
	return keys
}

func NewJSONFileCache(filePath string) (FileCache, error) {
	err := helpers.CreateDirAndFileIfNoExist(filePath)
	if err != nil {
		return nil, err
	}

	cache := &JSONFileCache{
		filePath: filePath,
	}

	return cache, nil
}

func (c *JSONFileCache) Store(key string, i interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	file := make(map[string]interface{})
	err := c.loadFullFile(&file)
	if err != nil {
		return err
	}

	file[key] = i

	err = c.saveToFile(file)
	if err != nil {
		return err
	}

	return nil
}

func (c *JSONFileCache) Get(key string, target interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	err := c.loadKeyFromFile(&target, key)
	if err != nil {
		return err
	}

	return nil
}

func (c *JSONFileCache) All() (FileCacheAllResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	items := make(map[string]interface{})
	err := c.loadFullFile(&items)
	if err != nil {
		return nil, err
	}

	result := createFileCacheAllResult(items)

	return result, nil
}

func (c *JSONFileCache) Purge() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data := make(map[string]interface{})
	err := c.saveToFile(data)
	if err != nil {
		return err
	}

	return nil
}

func (c *JSONFileCache) saveToFile(file map[string]interface{}) error {
	jsonData, err := json.Marshal(file)
	if err != nil {
		log.Error().Err(err).Msg("Error marshalling cache data")
		return err
	}

	err = os.WriteFile(c.filePath, jsonData, 0644)
	if err != nil {
		log.Error().Err(err).Msg("Error writing cache data to file")
		return err
	}

	return nil
}

func (c *JSONFileCache) loadFileBytes() ([]byte, error) {
	fileData, err := os.ReadFile(c.filePath)
	if err != nil {
		log.Error().Err(err).Msg("Error reading cache data from file")
		return nil, err
	}

	return fileData, nil
}

func (c *JSONFileCache) loadFullFile(target interface{}) error {
	fileData, err := c.loadFileBytes()
	if err != nil {
		return err
	}

	if err := json.Unmarshal(fileData, target); err != nil {
		return err
	}

	return nil
}

func (c *JSONFileCache) loadKeyFromFile(target interface{}, index string) error {
	file := make(map[string]interface{})
	err := c.loadFullFile(&file)
	if err != nil {
		return err
	}

	err = helpers.ExtractKeyFromMap(index, file, target)
	if err != nil {
		return err
	}

	return nil
}
