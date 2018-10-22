package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	cmap "github.com/orcaman/concurrent-map"
)

type value struct{}

type cache interface {
	store(string, *value)
	load(string) *value
	len() int64
}

type syncMap struct {
	mp sync.Map
}

func (c *syncMap) store(key string, v *value) {
	c.mp.Store(key, v)
}

func (c *syncMap) load(key string) *value {
	v, ok := c.mp.Load(key)
	if !ok {
		return nil
	}
	return v.(*value)
}

func (c *syncMap) len() int64 {
	var count int64
	c.mp.Range(func(k, v interface{}) bool {
		count++
		return true
	})
	return count
}

type cMap struct {
	mp cmap.ConcurrentMap
}

func (c *cMap) store(key string, v *value) {
	c.mp.Set(key, v)
}

func (c *cMap) load(key string) *value {
	v, ok := c.mp.Get(key)
	if !ok {
		return nil
	}
	return v.(*value)
}

func (c *cMap) len() int64 {
	return int64(c.mp.Count())
}

type rwMap struct {
	sync.RWMutex
	mp map[string]*value
}

func (c *rwMap) store(key string, v *value) {
	c.Lock()
	c.mp[key] = v
	c.Unlock()
}

func (c *rwMap) load(key string) *value {
	c.RLock()
	v, ok := c.mp[key]
	if !ok {
		c.RUnlock()
		return nil
	}
	c.RUnlock()
	return v
}

func (c *rwMap) len() int64 {
	c.RLock()
	res := len(c.mp)
	c.RUnlock()
	return int64(res)
}

func test(c cache, count int) {
	const pattern = "www.site%d.example.com.ua"
	wg := sync.WaitGroup{}
	now := time.Now()
	block := count / workers
	for i := 0; i < count; i += block {
		wg.Add(1)
		go func(from, to int) {
			for i := from; i < to; i++ {
				if i%10 == 0 {
					site := fmt.Sprintf(pattern, i)
					c.store(site, &value{})
				} else {
					c.load(fmt.Sprintf(pattern, rand.Intn(i)))
				}
			}
			wg.Done()
		}(i, i+block)
	}
	wg.Wait()
	end := time.Since(now)
	log.Printf("test compleat. duration:%s; len:%d", end, c.len())
}

const workers = 100

func main() {
	count := int(math.Pow10(6))
	for i := 0; i < 5; i++ {
		log.Println("testing concurrent-map")
		test(&cMap{mp: cmap.New()}, count)
		log.Println("testing sync-map")
		test(&syncMap{}, count)
		log.Println("testing map with rwmutex")
		test(&rwMap{mp: make(map[string]*value)}, count)
	}
}
