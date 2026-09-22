package utils

import (
	"math/rand"
	"time"
)

func SleepSec(seconds int) {
	sleepTime := time.Duration(seconds) * time.Second
	time.Sleep(sleepTime)
}

func SleepRND(limitSec int) {
	SleepRNDMin(0, limitSec)
}

// Sleeps a random duration in [minSec, limitSec) seconds.
func SleepRNDMin(minSec, limitSec int) {
	if limitSec <= 0 {
		return
	}
	if limitSec <= minSec {
		limitSec = minSec + 1
	}
	sleepTime := time.Duration(minSec+rand.Intn(limitSec-minSec)) * time.Second
	time.Sleep(sleepTime)
}

func SleepRNDMilliSec(minMilliSec, limitMilliSec int) {
	if limitMilliSec <= 0 {
		return
	}
	sleepTime := time.Duration(rand.Intn(limitMilliSec)+minMilliSec) * time.Millisecond
	time.Sleep(sleepTime)
}
