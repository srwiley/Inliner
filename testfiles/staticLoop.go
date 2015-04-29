// +build generate

package main

import ()

func runSingleLoopNotInlined() float64 {
	sum := 0.0
	for j := 0; j < 30; j++ {
		sum += float64(j*j/2 + j)
	}
	return sum
}

func runSingleLoop() float64 {
	sum := 0.0
	for j_ := 0; j_ < 30; j_++ {
		sum += float64(j_*j_/2 + j_)
	}
	return sum
}

func runDoubleLoopAsserts() (sum float64) {
	for j_ := 0; j_ < 5; j_++ {
		for k_ := 0; k_ < 4; k_++ {
			if k_%2 == 0 {
				sum += float64(j_ * k_)
			}
			affirm_(sum < 9)
		}
	}
	return sum
}

func runDoubleLoopNotInlined() float64 {
	sum := 0.0
	for j := 0; j < 3; j++ {
		for k := 0; k < 4; k++ {
			//fmt.Println(j_, k_)
			if k%2 == 0 {
				sum += float64(j * k)
			}
		}
	}
	return sum
}

func runDoubleLoop() float64 {
	sum := 0.0
	for j_ := 0; j_ < 3; j_++ {
		for k_ := 0; k_ < 4; k_++ {
			//fmt.Println(j_, k_)
			if k_%2 == 0 {
				sum += float64(j_ * k_)
			}
		}
	}
	return sum
}
