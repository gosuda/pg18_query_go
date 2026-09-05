package neoparse

//go:generate go run ./internal/genkeywords
//go:generate ragel -Z scan.rl -o scan.go
//go:generate go tool goyacc -o y.go -p yy gram.y
//go:generate gofmt -w scan.go y.go
