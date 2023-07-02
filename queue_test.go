package arch

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestQueue(t *testing.T) {
	q := NewQueue(20)
	wg := sync.WaitGroup{}
	type Pair struct {
		ID   int
		Time int
	}
	order := make([]*Pair, 100)
	for i := 0; i < len(order); i++ {
		q.push(i)
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			begin := time.Now()
			n := rand.Int() % 1000000000
			for i := 0; i < n; i++ {
				n--
			}
			q.set(i, i)
			order[i] = &Pair{
				ID:   i,
				Time: int(time.Since(begin).Milliseconds()),
			}
		}(i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			n := q.pop()
			fmt.Println(n)
		}()
	}
	wg.Wait()
	sort.Slice(order, func(i, j int) bool {
		return order[i].Time < order[j].Time
	})
	for i := 0; i < len(order); i++ {
		fmt.Println("ID: ", order[i].ID, " Time: ", order[i].Time)
	}
	fmt.Println("LEN ", len(order))
}
