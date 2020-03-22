// Package tests implements a suite of tests for certmagic.Storage
//
//
// Example Usage:
//
// package storage
//
// import (
//     tests "github.com/oyato/certmagic-storage-tests"
//     "testing"
// )
//
// func TestStorage(t *testing.T) {
//     // set up your storage
//     storage := NewInstanceOfYourStorage()
//     // then run the tests on it
//     tests.NewTestSuite(storage).Run(t)
// }
//
package tests

import (
	"bytes"
	"fmt"
	"github.com/caddyserver/certmagic"
	"math/rand"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"testing"
)

var (
	// KeyPrefix is prepended to all tested keys.
	// If changed, it must not contain a forward slash (/)
	KeyPrefix = "__test__key__"
)

// Suite implements tests for certmagic.Storage.
//
// Users should call Suite.Run() in their storage_test.go file.
type Suite struct {
	S   certmagic.Storage
	Rng interface{ Int() int }

	mu       sync.Mutex
	randKeys []string
}

// Run tests the Storage
//
// NOTE: t.Helper() is not called.
//       Test failure line numbers will be reported on files inside this package.
func (ts *Suite) Run(t *testing.T) {
	t.Cleanup(func() {
		ts.mu.Lock()
		defer ts.mu.Unlock()

		for _, k := range ts.randKeys {
			ts.S.Delete(k)
		}
	})
	ts.testLocker(t)
	ts.testStorageSingleKey(t)
	ts.testStorageDir(t)
}

func (ts *Suite) testLocker(t *testing.T) {
	key := strconv.Itoa(ts.Rng.Int())
	if err := ts.S.Unlock(key); err == nil {
		t.Fatalf("Storage successfully unlocks unlocked key")
	}
	if err := ts.S.Lock(key); err != nil {
		t.Fatalf("Storage fails to lock key: %s", err)
	}
	if err := ts.S.Unlock(key); err != nil {
		t.Fatalf("Storage fails to unlock locked key: %s", err)
	}

	test := func(key string) {
		for i := 0; i < 5; i++ {
			if err := ts.S.Lock(key); err != nil {
				// certmagic lockers can timeout
				continue
			}
			runtime.Gosched()
			if err := ts.S.Unlock(key); err != nil {
				t.Fatalf("Storage.Unlock failed: %s", err)
			}
		}
	}
	wg := &sync.WaitGroup{}
	for i := 0; i < 5; i++ {
		key := ts.randKey()
		for j := 0; j < 2; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				test(key)
			}()
		}
	}
	wg.Wait()
}

func (ts *Suite) testStorageSingleKey(t *testing.T) {
	key := ts.randKey()
	val := []byte(key)
	sto := ts.S
	sto.Lock(key)
	defer sto.Unlock(key)

	if sto.Exists(key) {
		t.Fatalf("Un-stored key %s exists", key)
	}

	if _, err := sto.Load(key); err == nil {
		t.Fatalf("Load(%s) should fail: the key was not stored", key)
	}

	if _, err := sto.Stat(key); err == nil {
		t.Fatalf("Stat(%s) should fail: the key doesn't exist", key)
	}

	if err := sto.Store("", []byte{}); err == nil {
		t.Fatalf("Store() with empty key should fail")
	}

	if err := sto.Store(key, nil); err != nil {
		t.Fatalf("Store(%s) with `nil` value failed: %s", key, err)
	}

	if err := sto.Store(key, []byte{}); err != nil {
		t.Fatalf("Store(%s) with empty value failed: %s", key, err)
	}

	if err := sto.Store(key, val); err != nil {
		t.Fatalf("Store(%s) failed: %s", key, err)
	}

	if !sto.Exists(key) {
		t.Fatalf("Stored key %s doesn't exists", key)
	}

	switch s, err := sto.Load(key); {
	case err != nil:
		t.Fatalf("Load(%s) failed: %s", key, err)
	case !bytes.Equal(val, s):
		t.Fatalf("Load(%s) failed: loaded %#v != stored %#v", key, s, val)
	}

	if err := sto.Delete(key); err != nil {
		t.Fatalf("Delete(%s) failed: %s", key, err)
	}

	if sto.Exists(key) {
		t.Fatalf("Deleted key still %s exists", key)
	}
}

func (ts *Suite) testStorageDir(t *testing.T) {
	sto := ts.S
	dir := ts.randKey()
	val := []byte(dir)
	k1 := dir + "/k1"
	k2 := dir + "/k/a/b"
	k3 := dir + "/k/c"
	ts.mu.Lock()
	ts.randKeys = append(ts.randKeys, k1, k2, k3)
	ts.mu.Unlock()

	if _, err := sto.List(k1, true); err == nil {
		t.Fatalf("List(%s, true) should fail: the key doesn't exist", k1)
	}

	if _, err := sto.List(k2, false); err == nil {
		t.Fatalf("List(%s, false) should fail: the key doesn't exist", k2)
	}

	if err := sto.Store(k1, val); err != nil {
		t.Fatalf("Store(%s) failed: %s", k1, err)
	}
	if err := sto.Store(k2, val); err != nil {
		t.Fatalf("Store(%s) failed: %s", k2, err)
	}
	if err := sto.Store(k3, val); err != nil {
		t.Fatalf("Store(%s) failed: %s", k3, err)
	}

	switch inf, err := sto.Stat(dir); {
	case err != nil:
		t.Fatalf("Stat(%s) failed: %s", dir, err)
	case inf.Key != dir:
		t.Fatalf("Stat(%s) failed: Key is set to %#v", dir, inf.Key)
	case inf.IsTerminal:
		t.Fatalf("Stat(%s) failed: IsTerminal should be false for directory keys", dir)
	}

	switch inf, err := sto.Stat(k2); {
	case err != nil:
		t.Fatalf("Stat(%s) failed: %s", k2, err)
	case inf.Key != k2:
		t.Fatalf("Stat(%s) failed: Key is set to %#v, but should be %#v", k2, inf.Key, k2)
	case !inf.IsTerminal:
		t.Fatalf("Stat(%s) failed: IsTerminal should be true for non-directory keys", k2)
	}

	if ls, err := sto.List(dir, false); err != nil {
		t.Fatalf("List(%s, false) failed: %s", dir, err)
	} else {
		sort.Strings(ls)
		got := fmt.Sprintf("%#v", ls)
		exp := fmt.Sprintf("%#v", []string{dir + "/k", k1})
		if got != exp {
			t.Fatalf("List(%s, false) failed: it should return %s, not %s", dir, exp, got)
		}
	}

	if ls, err := sto.List(dir, true); err != nil {
		t.Fatalf("List(%s, true) failed: %s", dir, err)
	} else {
		sort.Strings(ls)
		got := fmt.Sprintf("%#v", ls)
		exp := fmt.Sprintf("%#v", []string{
			dir + "/k",
			dir + "/k/a",
			dir + "/k/a/b",
			dir + "/k/c",
			k1,
		})
		if got != exp {
			t.Fatalf("List(%s, true) failed: it should return %s, not %s", dir, exp, got)
		}
	}
}

func (ts *Suite) randKey() string {
	return KeyPrefix + strconv.Itoa(ts.Rng.Int())
}

// NewTestSuite returns a new Suite initalised with storage s
// and a `rand.New(rand.NewSource(0))` random number generator
func NewTestSuite(s certmagic.Storage) *Suite {
	return &Suite{
		S:   s,
		Rng: rand.New(rand.NewSource(0)),
	}
}
