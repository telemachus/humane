.DEFAULT_GOAL := test

fmt:
	golangci-lint fmt --no-config -Egofumpt

staticcheck:
	staticcheck ./...

revive:
	revive -config revive.toml ./...

golangci:
	golangci-lint run

lint: staticcheck revive golangci

vuln:
	govulncheck ./...

test:
	go test -shuffle on ./...

testv:
	go test -shuffle on -v ./...

testr:
	go test -race -shuffle on ./...

build: lint testr
	go build ./...

clean:
	go clean
	rm -f bench-baseline.txt bench-new.txt

cacheclean:
	go clean -testcache

bench-quick:
	go test -bench='Basic|WithGroupChaining|WithAttrsChaining' -benchmem -benchtime=1s -count=3 -run=NONE

bench:
	go test -bench=. -benchmem -benchtime=1s -count=6 -run=NONE

bench-baseline:
	go test -bench=Humane -benchmem -benchtime=1s -count=6 -run=NONE > bench-baseline.txt

bench-compare:
	go test -bench=Humane -benchmem -benchtime=1s -count=6 -run=NONE > bench-new.txt
	benchstat bench-baseline.txt bench-new.txt

.PHONY: fmt staticcheck revive golangci lint vuln
.PHONY: test testv testr build clean cacheclean
.PHONY: bench-quick bench bench-baseline bench-compare
