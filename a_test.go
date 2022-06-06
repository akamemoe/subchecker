package main

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func dost(wg *sync.WaitGroup) {
	time.Sleep(time.Duration(rand.Intn(5)*100) * time.Millisecond)
	fmt.Println("doing..")
	wg.Done()

}
func TestA(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go dost(&wg)
		}
		wg.Wait()
		fmt.Println("loop", i)
	}
	fmt.Println("haha")

}
