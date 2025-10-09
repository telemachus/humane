.DEFAULT_GOAL := test

fmt:
	golangci-lint run --disable-all --no-config -Egofmt --fix
	golangci-lint run --disable-all --no-config -Egofumpt --fix

staticcheck: fmt
	staticcheck .

revive: fmt
	revive -config revive.toml .

golangci: fmt
	golangci-lint run

lint: fmt staticcheck revive golangci


build: lint
	go build .

install: build
	go install .

test:
	go test -shuffle on .

testr:
	go test -race -shuffle on .

testv:
	go test -shuffle on -v .

bench-quick:
	go test -bench='Basic|WithGroupChaining|WithAttrsChaining' -benchmem -benchtime=3s -count=3 -run=NONE

bench:
	go test -bench=. -benchmem -benchtime=5s -count=10 -run=NONE

bench-baseline:
	go test -bench=. -benchmem -benchtime=5s -count=10 -run=NONE > bench-baseline.txt

bench-compare:
	go test -bench=. -benchmem -benchtime=5s -count=10 -run=NONE > bench-new.txt
	benchstat bench-baseline.txt bench-new.txt

clean:
	go clean -i -r -cache

.PHONY: fmt staticcheck revive golangci lint build install test testv bench-quick bench bench-baseline bench-compare
