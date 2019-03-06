package main

import (
	"crypto/md5"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/ryanuber/go-glob"
)

// KeyNotFound type
type KeyNotFound struct {
	key string
}

// Error formats an error for the KeyNotFound type
func (e KeyNotFound) Error() string {
	return e.key + " " + "not found"
}

// KeyExpired type
type KeyExpired struct {
	Key string
}

// Error formats an error for the KeyExpired type
func (e KeyExpired) Error() string {
	return e.Key + " " + "expired"
}

// CacheIsFull type
type CacheIsFull struct {
}

// Error formats an error for the CacheIsFull type
func (e CacheIsFull) Error() string {
	return "Cache is Full"
}

// SerializerError type
type SerializerError struct {
}

// Error formats an error for the SerializerError type
func (e SerializerError) Error() string {
	return "Serializer error"
}

// Mesg represents a cache entry
type Mesg struct {
	Msg            *dns.Msg
	Blocked        bool
	LastUpdateTime time.Time
}

// Cache interface
type Cache interface {
	Get(key string) (Msg *dns.Msg, blocked bool, err error)
	Set(key string, Msg *dns.Msg, blocked bool) error
	Exists(key string) bool
	Remove(key string)
	Length() int
}

// MemoryCache type
type MemoryCache struct {
	Backend  map[string]*Mesg
	Maxcount int
	mu       sync.RWMutex
}

const (
	// BlockCacheEntryRegexp marks the regexp based BlockCache entries
	BlockCacheEntryRegexp = iota
	// BlockCacheEntryGlob marks the glob based BlockCache entries
	BlockCacheEntryGlob
)

// BlockCacheSpecial holds the extra data of a BlockCache entry
// used to perform glob or regexp matching.
type BlockCacheSpecial struct {
	Data string
	Type int
}

// MemoryBlockCache type
type MemoryBlockCache struct {
	Backend map[string]bool
	Special []BlockCacheSpecial
	mu      sync.RWMutex
}

// MemoryQuestionCache type
type MemoryQuestionCache struct {
	Backend  []QuestionCacheEntry `json:"entry"`
	Maxcount int
	mu       sync.RWMutex
}

// Get returns the entry for a key or an error
func (c *MemoryCache) Get(key string) (*dns.Msg, bool, error) {
	key = strings.ToLower(key)

	//Truncate time to the second, so that subsecond queries won't keep moving
	//forward the last update time without touching the TTL
	now := WallClock.Now().Truncate(time.Second)

	expired := false
	c.mu.Lock()
	mesg, ok := c.Backend[key]
	if ok && mesg.Msg == nil {
		ok = false
		logger.Warningf("Cache: key %s returned nil entry", key)
		c.removeNoLock(key)
	}
	if ok {
		elapsed := uint32(now.Sub(mesg.LastUpdateTime).Seconds())
		for _, answer := range mesg.Msg.Answer {
			if elapsed > answer.Header().Ttl {
				logger.Debugf("Cache: Key expired %s", key)
				c.removeNoLock(key)
				expired = true
			}
			answer.Header().Ttl -= elapsed
		}
	}
	c.mu.Unlock()

	if !ok {
		logger.Debugf("Cache: Cannot find key %s\n", key)
		return nil, false, KeyNotFound{key}
	}

	if expired {
		return nil, false, KeyExpired{key}
	}

	mesg.LastUpdateTime = now

	return mesg.Msg, mesg.Blocked, nil
}

// Set sets a keys value to a Mesg
func (c *MemoryCache) Set(key string, msg *dns.Msg, blocked bool) error {
	key = strings.ToLower(key)

	if c.Full() && !c.Exists(key) {
		return CacheIsFull{}
	}
	if msg == nil {
		logger.Debugf("Setting an empty value for key %s", key)
	}
	c.mu.Lock()
	c.Backend[key] = &Mesg{msg, blocked, WallClock.Now().Truncate(time.Second)}
	c.mu.Unlock()

	return nil
}

// Remove removes an entry from the cache
func (c *MemoryCache) removeNoLock(key string) {
	key = strings.ToLower(key)
	delete(c.Backend, key)
}

// Remove removes an entry from the cache
func (c *MemoryCache) Remove(key string) {
	c.mu.Lock()
	c.removeNoLock(key)
	c.mu.Unlock()
}

// Exists returns whether or not a key exists in the cache
func (c *MemoryCache) Exists(key string) bool {
	key = strings.ToLower(key)

	c.mu.RLock()
	_, ok := c.Backend[key]
	c.mu.RUnlock()
	return ok
}

// Length returns the caches length
func (c *MemoryCache) Length() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.Backend)
}

// Full returns whether or not the cache is full
func (c *MemoryCache) Full() bool {
	if c.Maxcount == 0 {
		return false
	}
	return c.Length() >= c.Maxcount
}

// KeyGen generates a key for the hash from a question
func KeyGen(q Question) string {
	h := md5.New()
	h.Write([]byte(q.String()))
	x := h.Sum(nil)
	key := fmt.Sprintf("%x", x)
	logger.Debugf("KeyGen: %s %s", q.String(), key)
	return key
}

// Get returns the entry for a key or an error
func (c *MemoryBlockCache) Get(key string) (bool, error) {
	key = strings.ToLower(key)

	c.mu.RLock()
	val, ok := c.Backend[key]
	c.mu.RUnlock()

	if !ok {
		return false, KeyNotFound{key}
	}

	return val, nil
}

// Remove removes an entry from the cache
func (c *MemoryBlockCache) Remove(key string) {
	key = strings.ToLower(key)

	c.mu.Lock()
	delete(c.Backend, key)
	c.mu.Unlock()
}

// Set sets a value in the BlockCache
func (c *MemoryBlockCache) Set(key string, value bool) error {
	key = strings.ToLower(key)
	const globChars = "?*"

	c.mu.Lock()
	if strings.ContainsAny(key, globChars) {
		c.Special = append(
			c.Special,
			BlockCacheSpecial{Data: key, Type: BlockCacheEntryGlob})
	} else {
		c.Backend[key] = value
	}
	c.mu.Unlock()

	return nil
}

// Exists returns whether or not a key exists in the cache
func (c *MemoryBlockCache) Exists(key string) bool {
	key = strings.ToLower(key)

	c.mu.RLock()
	_, ok := c.Backend[key]
	if !ok {
		for _, element := range c.Special {
			if element.Type == BlockCacheEntryRegexp {
				panic("Unsupported")
			} else if element.Type == BlockCacheEntryGlob {
				if glob.Glob(element.Data, key) {
					ok = true
				}
			}
		}
	}
	c.mu.RUnlock()
	return ok
}

// Length returns the caches length
func (c *MemoryBlockCache) Length() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.Backend)
}

// Add adds a question to the cache
func (c *MemoryQuestionCache) Add(q QuestionCacheEntry) {
	c.mu.Lock()
	if c.Maxcount != 0 && len(c.Backend) >= c.Maxcount {
		c.Backend = nil
	}
	c.Backend = append(c.Backend, q)
	c.mu.Unlock()
}

// Clear clears the contents of the cache
func (c *MemoryQuestionCache) Clear() {
	c.mu.Lock()
	c.Backend = make([]QuestionCacheEntry, 0, 0)
	c.mu.Unlock()
}

// Length returns the caches length
func (c *MemoryQuestionCache) Length() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.Backend)
}

// GetOlder eturns a slice of the entries older than `time`
func (c *MemoryQuestionCache) GetOlder(time int64) []QuestionCacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for i, e := range c.Backend {
		if e.Date > time {
			return c.Backend[i:]
		}
	}
	return []QuestionCacheEntry{}
}
