package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Develop a LRU Cache
// The cache will store Key/Value with expiration. If the expiration for key is set to 5 seconds,
// then that key should be evicted from the cache after 5 seconds. The cache can store maximum of
// 1024 keys.
// Must Haves
// ● Backend should be built on Golang
// ● The Get/set method in cache should be exposed as api endpoints
// Good to have
// ● Implementing concurrency in cache

// CacheItem represents an item in the cache with expiration time
type CacheItem struct {
	value      interface{}
	expiration int64
}

// Cache represents the cache structure
type Cache struct {
	items map[string]CacheItem
	mutex sync.RWMutex
}

// NewCache creates a new cache instance
func NewCache() *Cache {
	cache := &Cache{
		items: make(map[string]CacheItem),
	}
	go cache.startEvictionProcess()
	return cache
}

// new key-value pair to the cache with an expiration time
func (c *Cache) Set(key string, value interface{}, expiration time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.items[key] = CacheItem{
		value:      value,
		expiration: time.Now().Add(expiration).Unix(),
	}
}

// Get Method retrieves the value given key from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	item, found := c.items[key]
	if !found {
		return nil, false
	}
	if time.Now().Unix() > item.expiration {
		// Evict expired item
		delete(c.items, key)
		return nil, false
	}
	return item.value, true
}

// evicts expired items from the cache
func (c *Cache) evictExpiredItems() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for key, item := range c.items {
		if time.Now().Unix() > item.expiration {
			delete(c.items, key)
		}
	}
}

// startEvictionProcess starts a goroutine to periodically evict expired items from the cache
func (c *Cache) startEvictionProcess() {
	go func() {
		for {
			c.evictExpiredItems()
			time.Sleep(1 * time.Second) // Check every second for expired items
		}
	}()
}

func main() {

	cache := NewCache()

	//HTTP end Points and handlers
	http.HandleFunc("/get", cache.getHandler)
	http.HandleFunc("/set", cache.setHandler)

	// Start HTTP server
	fmt.Println("Server listening on port 8080")
	http.ListenAndServe(":8080", nil)
}

// Get the value
func (c *Cache) getHandler(w http.ResponseWriter, r *http.Request) {

	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest)
		return
	}

	value, ok := c.Get(key)
	if !ok {
		http.Error(w, "Key not found or expired", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(value)

}

// set the cache data
func (c *Cache) setHandler(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Key        string      `json:"key"`
		Value      interface{} `json:"value"`
		Expiration string      `json:"expiration"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	expiration, err := time.ParseDuration(data.Expiration)
	if err != nil {
		http.Error(w, "Invalid expiration duration", http.StatusBadRequest)
		return
	}
	c.Set(data.Key, data.Value, expiration) // Expiration set to 5 seconds
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Key %s set with value %s and expiration %s\n", data.Key, data.Value, expiration)
}