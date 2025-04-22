package main

import (
	"math/rand"
	"strconv"
)

func randNowithDash() string {

	randomInt := strconv.Itoa(rand.Intn(100))
	randomBigInt := strconv.Itoa(rand.Intn(1000))
	randomInfInt := strconv.Itoa(rand.Intn(5000))

	joinDashNo := "/" + randomInt + "-" + randomBigInt + "-" + randomInfInt
	return joinDashNo
}
