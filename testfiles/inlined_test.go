// test.go
package main

import (
	"errors"
	"fmt"
	"math"
	"testing"
)

func DenyErr(err error, t *testing.T) {
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}

func AffirmErr(err error, t *testing.T) {
	if err == nil {
		t.Error(err)
		t.FailNow()
	}
}

func TestNestedAssertInlines(t *testing.T) {
	sum := runDoubleLoopAsserts()
	if sum != 12 {
		err := errors.New("Sum not equal to 12 as expected,")
		DenyErr(err, t)
	}
	fmt.Println("TestNestedAssertInlines passed ", sum)
}

func TestNestedLoopedInlines(t *testing.T) {
	sum := compoundLoopedNotInlined()
	sumNoIn := compoundLoopedInline()
	sumNoInUnw := compoundLoopedInlineUnwound()
	delta := math.Abs(sum - sumNoIn)
	delta2 := math.Abs(sum - sumNoInUnw)
	if delta > 1e-12 || delta2 > 1e-12 {
		err := errors.New(fmt.Sprintln("Sums not equal as expected",
			sum, "vs", sumNoIn, "vs", sumNoInUnw, "delta", delta,
			"delta2", delta2))
		DenyErr(err, t)
	}
	fmt.Println("TestNestedLoopedInlines passed ", delta, delta2)
}

func TestNestedInlines(t *testing.T) {
	sum := compoundInline()
	sumNoIn := compoundNotInlined()
	delta := math.Abs(sum - sumNoIn)
	if delta > 1e-10 {
		err := errors.New(fmt.Sprintln("Sums not equal as expected",
			sum, "vs", sumNoIn, "delta", delta))
		DenyErr(err, t)
	}
	fmt.Println("TestNestedInlines passed ", delta)
}

func TestSingleLoop(t *testing.T) {
	sum := runSingleLoop()
	sumNoIn := runSingleLoopNotInlined()
	delta := math.Abs(sum - sumNoIn)
	if delta > 1e-15 {
		err := errors.New(fmt.Sprintln("Sums not equal as expected",
			sum, "vs", sumNoIn, " delta ", delta))
		DenyErr(err, t)
	}
	fmt.Println("TestSingleLoop passed ", delta)
}

func TestDoubleLoop(t *testing.T) {
	sum := runDoubleLoop()
	sumNoIn := runDoubleLoopNotInlined()
	delta := math.Abs(sum - sumNoIn)
	if delta > 1e-15 {
		err := errors.New(fmt.Sprintln("Sums not equal as expected",
			sum, "vs", sumNoIn, " delta ", delta))
		DenyErr(err, t)
	}
	fmt.Println("TestDoubleLoop passed ", delta)
}

func TestControlFlow(t *testing.T) {
	sum := runAssertFlowTest()
	//fmt.Println("tsum ", sum)
	if sum != 100 {
		err := errors.New("Sums not equal to 100 as expected")
		DenyErr(err, t)
	}
	fmt.Println("TestControlFlow passed ", sum)
}

func TestAssertNum(t *testing.T) {
	err := runAssertNumTest("5", "314") // Shoud pass
	DenyErr(err, t)
	err = runAssertNumTest("-5", "314") //This fails because the first number is negative
	AffirmErr(err, t)
	err = runAssertNumTest("589", "314") //This fails because the first number gt the second
	AffirmErr(err, t)
	err = runAssertNumTest("589.0", "314") //This fails because the first number is not an int
	AffirmErr(err, t)
	err = runAssertNumTest("589", "314.13") //This fails because the second number is not an int
	AffirmErr(err, t)
	err = runAssertNumTest("52", "314") // Should pass
	DenyErr(err, t)
	err = runAssertNumTest("523", "3144") // Should pass
	DenyErr(err, t)
	fmt.Println("TestAssertNum passed")
}

func Benchmark1_2xLocalNotInlined(b *testing.B) {
	for i := 0; i < b.N; i++ {
		compoundNotInlined()
	}
}

func Benchmark1_2xLocalInlined(b *testing.B) {
	for i := 0; i < b.N; i++ {
		compoundInline()
	}
}

func Benchmark2_2xLoopedNotInlined(b *testing.B) {
	for i := 0; i < b.N; i++ {
		compoundLoopedNotInlined()
	}
}

func Benchmark2_2xLoopedInlined(b *testing.B) {
	for i := 0; i < b.N; i++ {
		compoundLoopedInline()
	}
}

func Benchmark2_2xLoopedUnwound(b *testing.B) {
	for i := 0; i < b.N; i++ {
		compoundLoopedInlineUnwound()
	}
}

func Benchmark3_1xLoopNotUnwound(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runSingleLoopNotInlined()
	}
}

func Benchmark3_1xLoopUnwound(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runSingleLoop()
	}
}

func Benchmark4_2xLoopNotUnwound(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runDoubleLoopNotInlined()
	}
}

func Benchmark4_2xLoopUnwound(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runDoubleLoop()
	}
}
