package cache

import (
	"bytes"
	"errors"
	"hash/crc64"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/guionardo/gs-rproxy/internal/httpdata"
	"github.com/guionardo/gs-rproxy/internal/telemetry"
)

type (
	MemoryCache struct {
		maxLength uint
		items     map[uint64]*CacheItem
		totalSize uint64
		lock      sync.RWMutex
	}
	CacheItem struct {
		response   *httpdata.HttpResponse
		validUntil time.Time
		length     int
	}
)

func NewMemoryCache(maxLength uint) *MemoryCache {
	return &MemoryCache{
		maxLength: maxLength,
		items:     make(map[uint64]*CacheItem),
	}
}

func newCacheItem(resp *httpdata.HttpResponse, validuntil time.Time) *CacheItem {
	return &CacheItem{
		response:   resp,
		validUntil: validuntil,
		length:     int(resp.Length) + 8 + 8 + 8,
	}
}

var errNoWriteCache = errors.New("no")

func (c *MemoryCache) WriteCachedItem(r *http.Request, w http.ResponseWriter) error {
	if r.Method != http.MethodGet {
		return errNoWriteCache
	}
	c.lock.RLock()
	hash := requestHash(r)
	item, ok := c.items[hash]
	c.lock.RUnlock()
	if !ok {
		return errNoWriteCache
	}
	if time.Now().After(item.validUntil) {
		c.lock.Lock()
		delete(c.items, hash)
		c.lock.Unlock()
		return errNoWriteCache
	}
	w.WriteHeader(int(item.response.StatusCode))
	for h, values := range item.response.Headers {
		for _, value := range values {
			w.Header().Add(h, value)
		}
	}

	w.Write(item.response.Body)
	return nil
}

func (c *MemoryCache) SaveCachedItem(r *http.Request, resp *httpdata.HttpResponse) time.Time {
	if r.Method != http.MethodGet || resp.StatusCode >= 300 || resp.StatusCode < 200 {
		return time.Time{}
	}
	cacheControl := r.Header.Get("Cache-Control")
	if strings.Contains(cacheControl, "no-cache") {
		return time.Time{}
	}
	validUntil := time.Now().Add(time.Minute * 5) // Default cache TTL
	if strings.Contains(cacheControl, "max-age=") {
		maxAge, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(cacheControl, "max-age=")))
		if err == nil && maxAge > 0 {
			validUntil = time.Now().Add(time.Second * time.Duration(maxAge))
		}
	}
	hash := requestHash(r)
	c.lock.Lock()
	item := newCacheItem(resp, validUntil)
	c.items[hash] = item
	c.totalSize += uint64(item.length)
	c.lock.Unlock()
	c.invalidate()

	return validUntil
}

func (c *MemoryCache) invalidate() {
	c.lock.Lock()
	defer func() {
		telemetry.SetCacheLength(len(c.items))
		telemetry.SetCacheSize(c.totalSize)
		c.lock.Unlock()
	}()

	if len(c.items) <= int(c.maxLength) {
		return
	}
	for hash, item := range c.items {
		if item.validUntil.Before(time.Now()) {
			c.totalSize -= uint64(item.length)
			delete(c.items, hash)
		}
	}
	if len(c.items) <= int(c.maxLength) {
		return
	}
	// TODO: Implementar reducao do cache removendo os mais antigos

	tmp := make([]tmpInv, len(c.items))
	index := 0
	for hash, item := range c.items {
		tmp[index] = tmpInv{hash, item}
	}
	sort.Slice(tmp, func(i, j int) bool {
		return tmp[i].item.validUntil.After(tmp[j].item.validUntil)
	})
	c.items = make(map[uint64]*CacheItem, c.maxLength)
	totalSize := 0
	for index := range int(c.maxLength) {
		c.items[tmp[index].hash] = tmp[index].item
		totalSize += tmp[index].item.length
	}
	c.totalSize = uint64(totalSize)
}

type tmpInv struct {
	hash uint64
	item *CacheItem
}

func requestHash(r *http.Request) uint64 {
	w := bytes.NewBufferString(r.URL.String())
	r.Header.Write(w)
	return crc64.Checksum(w.Bytes(), crc64.MakeTable(crc64.ECMA))
}
