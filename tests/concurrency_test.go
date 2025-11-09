package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/PetarGeorgiev-hash/flashdb/store"
)

func TestConcurrentSetAndGet(t *testing.T) {
	s := store.NewStore()
	defer s.Close()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "k" + string(rune(i))
			s.Set(key, []byte("value"), time.Second*5)
			s.Get(key)
		}(i)
	}
	wg.Wait()
}
