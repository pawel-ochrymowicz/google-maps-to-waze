-include .env

SOURCE_FILES?=./...
TEST_PATTERN?=.
TEST_OPTIONS?=

export GOLANGCI_LINT_VERSION := v1.52.2
export GO111MODULE := on
export GOFLAGS := -mod=vendor

bin/golangci-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s $(GOLANGCI_LINT_VERSION)

run_poll:
	TELEGRAM_TOKEN=$(TELEGRAM_TOKEN) go run ./app/main.go

lint: bin/golangci-lint
	./bin/golangci-lint run ./...

test:
	go test $(TEST_OPTIONS) -failfast -race -coverpkg=./... -covermode=atomic $(SOURCE_FILES) -run $(TEST_PATTERN) -timeout=2m -count=1

build:
	go build -race -o bin/google-maps-to-waze ./app

ci: test lint build

docker_build:
	docker build -t europe-west1-docker.pkg.dev/disco-abacus-385513/google-maps-to-waze/google-maps-to-waze:latest --progress=plain .

docker_push:
	docker push europe-west1-docker.pkg.dev/disco-abacus-385513/google-maps-to-waze/google-maps-to-waze:latest

.PHONY: run_poll lint test build ci docker_build docker_push