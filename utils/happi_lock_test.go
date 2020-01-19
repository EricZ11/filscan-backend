package utils

import (
	"fmt"
	"testing"
	"time"
)

func TestX(t *testing.T) {
	lock := NewHappiLock(true)

	if l, isok := lock.(*Sexylock); isok {
		l.is_debug = true
	}

	lock.Lock()
	lock.Lock()
	go func() {
		lock.Lock()
		defer lock.Unlock()
		fmt.Printf("in goroutine function___1!!!\n")
	}()
	go func() {
		lock.Lock()
		defer lock.Unlock()
		fmt.Printf("in goroutine function___2!!!\n")
	}()
	time.Sleep(time.Second)
	lock.Unlock()
	time.Sleep(time.Second)
	lock.Unlock()

	print("wait all lock release......sleeping!\n\n")
	time.Sleep(time.Second)
}
