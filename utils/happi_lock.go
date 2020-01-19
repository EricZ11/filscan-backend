package utils

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type Sexylock struct {
	mu       *sync.Mutex
	inermutx *sync.Mutex
	owner    int
	is_debug bool
}

func NewHappiLock(is_debug bool) sync.Locker {
	rl := &Sexylock{}

	rl.mu = new(sync.Mutex)
	rl.inermutx = new(sync.Mutex)

	rl.is_debug = is_debug
	return rl
}

func GetGoroutineId() int {
	defer func() {
		if err := recover(); err != nil {
			panic(fmt.Sprintf("panic recover:panic info:%v\n", err))
		}
	}()

	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Sprintf("cannot get goroutine id: %v", err))
	}
	return id
}

func (rl *Sexylock) printf(fmts string, args ...interface{}) {
	if rl.is_debug {
		Printf("happilock", fmts, args[:]...)
	}
}

func (rl *Sexylock) is_owner(id int) bool {
	rl.inermutx.Lock()
	defer rl.inermutx.Unlock()
	return rl.owner == id
}

func (rl *Sexylock) set_owner(id int) {
	rl.inermutx.Lock()
	defer rl.inermutx.Unlock()
	rl.owner = id
}

func (rl *Sexylock) is_locked() bool {
	return !rl.is_owner(0)
}

func (rl *Sexylock) Lock() {
	me := GetGoroutineId()

	var is_owner, is_locked bool
	rl.inermutx.Lock()
	is_owner = rl.owner == me
	is_locked = rl.owner != 0
	rl.inermutx.Unlock()

	if is_owner {
		rl.printf("YES!!!! I (%d) have locked resource, just return!\n", rl.owner)
		return
	}

	if is_locked {
		rl.printf("the resouce was locked by %d, %d want a lock, so wating...\n", rl.owner, me)
	}

	rl.mu.Lock()
	rl.owner = me
	rl.printf("i(%d) first locked just return\n", rl.owner)
}

func (rl *Sexylock) Unlock() {
	me := GetGoroutineId()

	var is_owner, is_locked bool

	rl.inermutx.Lock()
	is_owner = rl.owner == me
	is_locked = rl.owner != 0
	if is_locked && !is_owner {
		rl.printf("warning: resouce is locked by:%d, please take attention on this warnning\n", rl.owner)
	}
	if is_locked {
		rl.printf("i(%d) will unlock resouce\n", rl.owner)
		rl.owner = 0
		rl.mu.Unlock()
	}
	rl.inermutx.Unlock()
}
