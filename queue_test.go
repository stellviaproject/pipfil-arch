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
	q.run(func(v any) {
		n := v.(int)
		fmt.Println(n)
	})
	for i := 0; i < len(order); i++ {
		ch := q.push(i)
		wg.Add(1)
		go func(i int, ch chan any) {
			defer wg.Done()
			begin := time.Now()
			n := rand.Int() % 1000000000
			for i := 0; i < n; i++ {
				n--
			}
			order[i] = &Pair{
				ID:   i,
				Time: int(time.Since(begin).Milliseconds()),
			}
			ch <- i
			q.set()
		}(i, ch)
	}
	wg.Wait()
	q.exit()
	sort.Slice(order, func(i, j int) bool {
		return order[i].Time < order[j].Time
	})
	for i := 0; i < len(order); i++ {
		fmt.Println("ID: ", order[i].ID, " Time: ", order[i].Time)
	}
	fmt.Println("LEN ", len(order))
}
