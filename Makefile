.DEFAULT_GOAL := test

fmt:
	golangci-lint fmt --no-config -Egofumpt

# Pinned to go1.27.1: staticcheck can't read go1.27.2's export data yet.
# Remove once a staticcheck release supports it.
staticcheck:
	GOTOOLCHAIN=go1.27.1 staticcheck ./...

revive:
	revive -config revive.toml ./...

golangci:
	golangci-lint run

lint: staticcheck revive golangci

build: lint
	go build ./...

test:
	go test -shuffle on ./...

testr:
	go test -race -shuffle on ./...

testv:
	go test -shuffle on -v ./...

clean:
	go clean
	rm -f bench-baseline.txt bench-new.txt

bench-quick:
	go test -bench='Basic|WithGroupChaining|WithAttrsChaining' -benchmem -benchtime=1s -count=3 -run=NONE

bench:
	go test -bench=. -benchmem -benchtime=1s -count=6 -run=NONE

bench-baseline:
	go test -bench=Humane -benchmem -benchtime=1s -count=6 -run=NONE > bench-baseline.txt

bench-compare:
	go test -bench=Humane -benchmem -benchtime=1s -count=6 -run=NONE > bench-new.txt
	benchstat bench-baseline.txt bench-new.txt

.PHONY: fmt staticcheck revive golangci lint
.PHONY: test testv testr
.PHONY: bench-quick bench bench-baseline bench-compare
.PHONY: build clean
