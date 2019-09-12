package __6

import "sync"

//Clock : structure counting received transactions
type Clock struct {
	m    map[string][]int
	lock sync.Mutex
}

//MakeClock : Clock initiator
func MakeClock() *Clock {
	clock := new(Clock)
	clock.m = make(map[string][]int)
	return clock
}

//setClock : increments Clock
func setClock(id string, packageNumber int) {
	tClock.m[id] = append(tClock.m[id], packageNumber)
}

//checkClock : Checks if a packageNumber has been received from an myID before
func checkClock(id string, packageNumber int) bool {
	if contains(tClock.m[id], packageNumber) { //Checks if packageNumber was seen
		return true
	} else {
		return false
	}
}

//logic for checkClock
func contains(c []int, n int) bool {
	for _, x := range c {
		if x == n {
			return true
		}
	}
	return false
}
