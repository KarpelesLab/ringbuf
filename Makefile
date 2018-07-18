#!/bin/make

all:
	goimports -w -l .
	go build -v

test:
	goimports -w -l .
	go test

cover:
	goimports -w -l .
	go test -cover -coverprofile=c.out
	go tool cover -html=c.out -o coverage.html
