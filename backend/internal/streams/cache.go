package streams

import (
	"net/http"
	"sync"
	"time"
)

// pathListCache caches the mediamtx /v3/paths/list response for a short TTL,
// eliminating N mediamtx API calls per dashboard load (one call serves all cameras).
type pathListCache struct {
	mu        sync.RWMutex
	paths     map[string]pathState
	fetchedAt time.Time
	ttl       time.Duration
}

func newPathListCache(ttl time.Duration) *pathListCache {
	return &pathListCache{ttl: ttl}
}

// get returns a copy of the cached path map, fetching fresh data when stale.
func (c *pathListCache) get(client *http.Client) (map[string]pathState, error) {
	// Fast path: cache is fresh.
	c.mu.RLock()
	if c.paths != nil && time.Since(c.fetchedAt) < c.ttl {
		result := copyPaths(c.paths)
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	// Slow path: acquire write lock and re-check before fetching.
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.paths != nil && time.Since(c.fetchedAt) < c.ttl {
		return copyPaths(c.paths), nil
	}

	paths, err := listPaths(client)
	if err != nil {
		// Return stale data if available rather than an error.
		if c.paths != nil {
			return copyPaths(c.paths), nil
		}
		return nil, err
	}
	c.paths = paths
	c.fetchedAt = time.Now()
	return copyPaths(paths), nil
}

// invalidate forces the next get() to re-fetch from mediamtx.
func (c *pathListCache) invalidate() {
	c.mu.Lock()
	c.fetchedAt = time.Time{}
	c.mu.Unlock()
}

func copyPaths(m map[string]pathState) map[string]pathState {
	out := make(map[string]pathState, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
