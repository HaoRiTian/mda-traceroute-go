package mda

import "time"

type MatchCache struct {
	Cache       *SyncMap
	FlowIDCache *SyncMap
}

func NewMatchCache(hash string, timeout, checkFreq uint8) *MatchCache {
	to := time.Duration(timeout) * time.Second
	cf := time.Duration(checkFreq) * time.Second
	c := &MatchCache{
		Cache:       NewSyncMap(hash, to, cf),
		FlowIDCache: NewSyncMap(hash, to, cf),
	}
	return c
}

func (mc *MatchCache) Close() {
	mc.Cache.Close()
}
