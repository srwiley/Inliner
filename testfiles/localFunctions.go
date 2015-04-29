// +build generate

package main

import ()

var sumG = 0.0

func inlineTestG_(x float64, y float64) {
	sumG /= x + y
	sumG *= x * y
}

func compoundInline() float64 {
	sum := 0.0
	inlineTest_ := func(x float64, y float64) {
		sum += x * y
		sum += x - y
	}
	inlineTest2_ := func(x float64, y float64) {
		sum += x + y
		sum += x / y
		inlineTest_(x/3.2+y, y)
	}
	inlineTest3_ := func(x float64, y float64) {
		sum += x/2 + y/3
		inlineTest2_(x+9.2, y)
	}
	inlineTest2_(4.3+3.2, 2.0)
	inlineTest3_(4.3, 2.4)
	inlineTest3_(5.3-38.2, 2.74-9.4)
	inlineTest3_(4.6, 7.4)
	inlineTest3_(30.2, 92.4)
	inlineTestG_(30.2, 92.4)
	return sum
}

func compoundNotInlined() float64 {
	sum := 0.0
	inlineTest := func(x float64, y float64) {
		sum += x * y
		sum += x - y
	}
	inlineTest2 := func(x float64, y float64) {
		sum += x + y
		sum += x / y
		inlineTest(x/3.2+y, y)
	}
	inlineTest3 := func(x float64, y float64) {
		sum += x/2 + y/3
		inlineTest2(x+9.2, y)
	}
	inlineTest2(4.3+3.2, 2.0)
	inlineTest3(4.3, 2.4)
	inlineTest3(5.3-38.2, 2.74-9.4)
	inlineTest3(4.6, 7.4)
	inlineTest3(30.2, 92.4)
	return sum
}

func compoundLoopedNotInlined() float64 {
	sum := 0.0
	inlineTest := func(x float64, y float64) {
		sum += x * y
		sum += x - y
	}
	inlineTest2 := func(x float64, y float64) {
		sum += x + y
		sum += x / y
		inlineTest(x/3.2+y, y)
	}
	inlineTest3 := func(x float64, y float64) {
		sum += x/2 + y/3
		inlineTest2(x+9.2, y)
	}
	for i := 0; i < 50; i++ {
		if i%2 == 0 {
			inlineTest3(45.2, 4.2-float64(i))
		}
	}
	return sum
}

func compoundLoopedInline() float64 {
	sum := 0.0
	inlineTest_ := func(x float64, y float64) {
		sum += x * y
		sum += x - y
	}
	inlineTest2_ := func(x float64, y float64) {
		sum += x + y
		sum += x / y
		inlineTest_(x/3.2+y, y)
	}
	inlineTest3_ := func(x float64, y float64) {
		sum += x/2 + y/3
		inlineTest2_(x+9.2, y)
	}
	for i := 0; i < 50; i++ {
		if i%2 == 0 {
			inlineTest3_(45.2, 4.2-float64(i))
		}
	}
	return sum
}

func compoundLoopedInlineUnwound() float64 {
	sum := 0.0
	inlineTest_ := func(x float64, y float64) {
		sum += x * y
		sum += x - y
	}
	inlineTest2_ := func(x float64, y float64) {
		sum += x + y
		sum += x / y
		inlineTest_(x/3.2+y, y)
	}
	inlineTest3_ := func(x float64, y float64) {
		sum += x/2 + y/3
		inlineTest2_(x+9.2, y)
	}
	for i_ := 0; i_ < 50; i_++ { // Ensure subsitutions work in sub-blocks
		if i_%2 == 0 {
			inlineTest3_(45.2, 4.2-float64(i_))
		}
	}
	return sum
}
