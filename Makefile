-include .env

SOURCE_FILES?=./...
TEST_PATTERN?=.
TEST_OPTIONS?=

export GOLANGCI_LINT_VERSION := v1.61.0
export GO111MODULE := on
export GOFLAGS := -mod=vendor
export LOCAL_SERVER_PORT := 8888

bin/golangci-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s $(GOLANGCI_LINT_VERSION)

lint: bin/golangci-lint
	./bin/golangci-lint run ./...

test:
	go test $(TEST_OPTIONS) -failfast -race -coverpkg=./... -covermode=atomic $(SOURCE_FILES) -run $(TEST_PATTERN) -timeout=2m -count=1

build:
	go build -race -o bin/ $(SOURCE_FILES)

ci: test lint build

docker_build:
	docker build -t $(IMAGE_NAME):latest --progress=plain .

docker_push:
	docker push $(IMAGE_NAME):latest

run_poll: build
	TELEGRAM_TOKEN=$(TELEGRAM_TOKEN) bin/app

setup_ngrok:
	@echo "Starting Ngrok..."
	@ngrok http $(LOCAL_SERVER_PORT) > /dev/null & \
	sleep 5; \
	NGROK_URL=$(curl --silent --max-time 10 http://127.0.0.1:4040/api/tunnels | grep -o '"public_url":"[^"]*' | head -1 | cut -d'"' -f4); \
	UUID=$$(uuidgen); \
	export TELEGRAM_WEBHOOK_LINK="$$NGROK_URL/$$UUID"; \
	echo "Your TELEGRAM_WEBHOOK_LINK is: $$TELEGRAM_WEBHOOK_LINK"; \

shutdown_ngrok:
	@echo "Shutting down Ngrok..."
	@pkill ngrok

run_webhook: build
	TELEGRAM_TOKEN=$(TELEGRAM_TOKEN) TELEGRAM_WEBHOOK_LINK=$(TELEGRAM_WEBHOOK_LINK) bin/app

.PHONY: run_poll lint test build ci docker_build docker_push setup_ngrok shutdown_ngrok run_webhook