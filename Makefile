.DEFAULT_GOAL := test

fmt:
	golangci-lint run --disable-all --no-config -Egofmt --fix
	golangci-lint run --disable-all --no-config -Egofumpt --fix

revive: fmt
	revive -config .revive.toml

lint: revive
	golangci-lint run

build: lint
	go build .

install: build
	go install .

test:
	go test -shuffle on .

testv:
	go test -shuffle on -v .

bench:
	go test -bench=. -benchmem -benchtime=5s -count=3 -run=NONE

.PHONY: fmt revive lint build install test testv bench
