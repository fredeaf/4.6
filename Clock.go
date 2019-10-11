package main

import "sync"

//Clock : structure registering received transactions
type Clock struct {
	m    map[string][]string
	lock sync.Mutex
}

//MakeClock : Clock initiator
func MakeClock() *Clock {
	clock := new(Clock)
	clock.m = make(map[string][]string)
	return clock
}

//setClock : increments Clock
func setClock(id string, packageNumber string) {
	tClock.m[id] = append(tClock.m[id], packageNumber)
}

//checkClock : Checks if a packageNumber is new
func checkClock(id string, packageNumber string) bool {
	if contains(tClock.m[id], packageNumber) { //Checks if packageNumber was seen
		return false
	} else {
		return true
	}
}

//logic for checkClock
func contains(c []string, n string) bool {
	for _, x := range c {
		if x == n {
			return true
		}
	}
	return false
}
