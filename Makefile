.PHONY: build vet test run-dev

build:
	go build ./...

vet:
	go vet ./...

test:
	go test ./...

run-dev:
	./scripts/run-dev.sh
