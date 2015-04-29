package main

//go:generate -command inline ../inliner
//go:generate inline -out asserts_inlined.go -in asserts.go
//go:generate gofmt -w=true asserts_inlined.go
//go:generate inline -out localFunctions_inlined.go -in localFunctions.go
//go:generate gofmt -w=true localFunctions_inlined.go
//go:generate inline -out staticLoop_inlined.go -in staticLoop.go
//go:generate gofmt -w=true staticLoop.go

func main() {
	runDoubleLoop()
}
