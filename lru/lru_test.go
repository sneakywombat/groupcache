/*
Copyright 2013 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package lru

import (
	"fmt"
	"testing"
	"time"
)

type simpleStruct struct {
	int
	string
}

type complexStruct struct {
	int
	simpleStruct
}

var getTests = []struct {
	name       string
	keyToAdd   interface{}
	keyToGet   interface{}
	expectedOk bool
	refreshCnt int
}{
	{"string_hit", "myKey10", "myKey10", true, 0},
	{"string_miss", "myKey20", "nonsense", false, 0},
	{"simple_struct_hit", simpleStruct{1, "two"}, simpleStruct{1, "two"}, true, 0},
	{"simple_struct_miss", simpleStruct{1, "two"}, simpleStruct{0, "noway"}, false, 0},
	{"complex_struct_hit", complexStruct{1, simpleStruct{2, "three"}},
		complexStruct{1, simpleStruct{2, "three"}}, true, 0},
}

func TestGet(t *testing.T) {
	for _, tt := range getTests {
		lru := New(0)
		lru.Add(tt.keyToAdd, 1234, 0)
		val, ok := lru.Get(tt.keyToGet)
		if ok != tt.expectedOk {
			t.Fatalf("%s: cache hit = %v; want %v", tt.name, ok, !ok)
		} else if ok && val != 1234 {
			t.Fatalf("%s expected get to return 1234 but got %v", tt.name, val)
		}
	}
}

func TestRemove(t *testing.T) {
	lru := New(0)
	lru.Add("myKey", 1234, 0)
	if val, ok := lru.Get("myKey"); !ok {
		t.Fatal("TestRemove returned no match")
	} else if val != 1234 {
		t.Fatalf("TestRemove failed.  Expected %d, got %v", 1234, val)
	}

	lru.Remove("myKey")
	if _, ok := lru.Get("myKey"); ok {
		t.Fatal("TestRemove returned a removed entry")
	}
}

func TestEvict(t *testing.T) {
	evictedKeys := make([]Key, 0)
	onEvictedFun := func(key Key, value interface{}) {
		evictedKeys = append(evictedKeys, key)
	}

	lru := New(20)
	lru.OnEvicted = onEvictedFun
	for i := 0; i < 22; i++ {
		lru.Add(fmt.Sprintf("myKey%d", i), 1234, 0)
	}

	if len(evictedKeys) != 2 {
		t.Fatalf("got %d evicted keys; want 2", len(evictedKeys))
	}
	if evictedKeys[0] != Key("myKey0") {
		t.Fatalf("got %v in first evicted key; want %s", evictedKeys[0], "myKey0")
	}
	if evictedKeys[1] != Key("myKey1") {
		t.Fatalf("got %v in second evicted key; want %s", evictedKeys[1], "myKey1")
	}
}

func TestGetValidity(t *testing.T) {
	onExpiredFun := func(key Key, value interface{}) (Key, interface{}) {
		switch key {
		case "myKey10", "myKey20":
			return key, value
		}
		return nil, nil
	}

	lru := New(0)
	lru.OnExpired = onExpiredFun
	for _, tt := range getTests {
		switch tt.keyToAdd {
		case "myKey10", "myKey20":
			lru.Add(tt.keyToAdd, 1234, 500*time.Millisecond)
		default:
			lru.Add(tt.keyToAdd, 1234, 0)
		}
	}
	// wait for our two keys above to expire out
	time.Sleep(2 * time.Second)
	cacheLen := lru.Len()
	if cacheLen != 4 {
		t.Fatalf("lru size wrong, too many keys expired; want 3 items in lru, got %v", cacheLen)
	}
	// remove the autorefreshed keys now
	lru.Remove("myKey10")
	lru.Remove("myKey20")
	for _, tt := range getTests {
		_, ok := lru.Get(tt.keyToGet)
		switch tt.keyToAdd {
		case "myKey10", "myKey20":
			if ok {
				t.Fatalf("%v should have expired, but remains", tt.keyToGet)
			}
		default:
			if ok != tt.expectedOk {
				t.Fatalf("%s:%v cache hit = %v; want %v", tt.name, tt.keyToGet, ok, !ok)
			}
		}
	}
}
