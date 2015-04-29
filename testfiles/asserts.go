// +build generate

package main

import (
	"errors"
	"strconv"
)

// The expected sum is 100 after all the breaks and continues
func runAssertFlowTest() (sum int) {
	var numStr []string
	sum = 5
loop:
	for i := 0; i < 16; i++ {
		deny_(i == 10, "continue")
		numStr = append(numStr, strconv.Itoa(i))
		affirm_(numStr)
		deny_(i == 14, "break loop")
	}
	//fmt.Println("len nums ", len(numStr))
	affirm_(len(numStr) == 14)
	for _, s := range numStr {
		n, err := strconv.Atoi(s)
		deny_(err, "break")
		affirm_(n >= 0)
		sum += n
	}
	return
}

// An error is returned it either number string is not parsable as an int
// or n1 >= n2 or n1 <= 0
func runAssertNumTest(numStr1, numStr2 string) (rerr error) {
	number, err := strconv.Atoi(numStr1)
	deny_(err, `return err`) // Return if err is not nil
	errNgt := errors.New("number not gt 0")
	affirm_(number > 0, `return errNgt`) // Return if number is not gt zero
	number2, err2 := strconv.Atoi(numStr2)
	deny_(err2, `return err2`)                          // Return if err is not nil
	errNum := errors.New("number2 greater than number") // Return if number2 > number
	affirm_(number2 > number, `return errNum`)
	deny_(err2, `rerr = err2; return`)
	return
}
