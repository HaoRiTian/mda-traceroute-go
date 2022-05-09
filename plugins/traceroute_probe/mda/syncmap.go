package mda

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"mda-traceroute-go/util"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// SyncMap 处理实时数据流
type SyncMap struct {
	Name       string
	Data       sync.Map
	Timeout    int64
	CheckFreq  int64
	ExpireTime sync.Map
	exit       uint32
	keys       []uint32 // 排序插入

	Repeat bool // 标记是否有相同key对应同一个val，专门为判断负载均衡类型设定的
}

// NewSyncMap is a construct function to create syncmap.
func NewSyncMap(name string, timeout time.Duration, checkfreq time.Duration) *SyncMap {
	t := int64(timeout)
	f := int64(checkfreq)

	return &SyncMap{
		Name:      name,
		Timeout:   t,
		CheckFreq: f,
		exit:      0,
		keys:      make([]uint32, 256),
		Repeat:    false,
	}
}

//Load returns the value from syncmap
func (m *SyncMap) Load(key interface{}) (value interface{}, ok bool) {
	return m.Data.Load(key)
}

type keyjson struct {
	Key string
}

func (m *SyncMap) LoadRestApi(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	v := new(keyjson)
	_ = json.NewDecoder(r.Body).Decode(v)

	if v.Key == "internal_fetch_keylist" {
		var result []interface{}
		m.Data.Range(func(k, v interface{}) bool {
			result = append(result, k)
			return true
		})
		json.NewEncoder(w).Encode(result)
	} else {
		result, find := m.Data.Load(v.Key)
		if find {
			json.NewEncoder(w).Encode(result)
		} else {
			json.NewEncoder(w).Encode("")

		}
	}
}

func (m *SyncMap) GetRemainTime(key interface{}) (time.Duration, error) {
	exp, ok := m.ExpireTime.Load(key)
	if ok {
		remainTime := exp.(time.Time).Sub(time.Now())
		return remainTime, nil
	}
	return 0 * time.Second, fmt.Errorf("key does not exist")
}

//Store is used save the key,value pairs in syncmap
func (m *SyncMap) Store(key interface{}, value interface{}, currentTime time.Time) {
	//Check ExpireTime Map.
	exp, ok := m.ExpireTime.Load(key)
	if !ok {
		expireTime := currentTime.Add(time.Duration(m.Timeout))
		m.ExpireTime.Store(key, expireTime)
	} else {
		m.Repeat = true
		elapsed := exp.(time.Time).Sub(currentTime)
		//elapsed time less than half of timeout, update ExpireTime Store.
		if elapsed < time.Duration(m.Timeout/2) {
			expireTime := currentTime.Add(time.Duration(m.Timeout))
			m.ExpireTime.Store(key, expireTime)
		}
	}
	m.Data.Store(key, value)
	//m.keys = append(m.keys, key.(uint32))
	util.SortInsertUint32(&m.keys, key.(uint32))
}

//UpdateTime is used update specific key's expiretime.
func (m *SyncMap) UpdateTime(key interface{}, currentTime time.Time) {
	expireTime := currentTime.Add(time.Duration(m.Timeout))
	m.ExpireTime.Store(key, expireTime)
}

func (m *SyncMap) Delete(key interface{}) {
	m.Data.Delete(key)
	m.ExpireTime.Delete(key)

	for i, _ := range m.keys {
		if m.keys[i] == key.(uint32) {
			m.keys = append(m.keys[:i], m.keys[i+1:]...)
			break
		}
	}
}

// RunCheck 检查map中数据是否存在超时，检查间隔为 CheckFreq
func (m *SyncMap) RunCheck() {
	rand.Seed(time.Now().UnixNano())
	r := m.CheckFreq / 5
	for {
		if atomic.LoadUint32(&m.exit) != 0 {
			return
		}

		currentTime := time.Now()
		m.ExpireTime.Range(func(k, v interface{}) bool {
			value := v.(time.Time)
			if value.Sub(currentTime) < 0 {
				m.Delete(k)
			}
			return true
		})
		time.Sleep(time.Duration(m.CheckFreq + rand.Int63n(r)))
	}
}

func (m *SyncMap) Len() int {
	return len(m.keys)
}

// Close 停止 RunCheck
func (m *SyncMap) Close() {
	atomic.StoreUint32(&m.exit, 1)
}

func (m *SyncMap) String() string {
	return fmt.Sprintf("{Name: %s, Data: %v, Timeout: %d, CheckFreq: %d, ExpireTime: %v}", m.Name, m.Data, m.Timeout, m.CheckFreq, m.ExpireTime)
}
