export PATH := /usr/local/go/bin:$(PATH)

.PHONY: build test test-coverage lint vet fmt clean run install web-build \
	trial-up trial-down test-infra-up test-infra-down test-integration

# Default target
run:
	go run .

web-build:
	cd web && npm ci && npm run build

build: web-build
	go build -o bin/reloquent .

test:
	go test ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	@echo "To view coverage report: go tool cover -html=coverage.out"

lint:
	golangci-lint run

vet:
	go vet ./...

fmt:
	gofmt -s -w .

clean:
	rm -rf bin/ dist/ web/dist/ web/node_modules/

install:
	go install .

# Trial mode
trial-up:
	docker compose -f trial/docker-compose.yml up --build -d
	@echo "Open http://localhost:8230"

trial-down:
	docker compose -f trial/docker-compose.yml down -v

# Integration tests
test-infra-up:
	docker compose -f test/docker-compose.test.yml up -d --wait

test-infra-down:
	docker compose -f test/docker-compose.test.yml down -v

test-integration: test-infra-up
	RELOQUENT_TEST_PG_HOST=localhost \
	RELOQUENT_TEST_PG_PORT=25432 \
	RELOQUENT_TEST_PG_DATABASE=reloquent_test \
	RELOQUENT_TEST_PG_USER=postgres \
	RELOQUENT_TEST_PG_PASSWORD=postgres \
	RELOQUENT_TEST_MONGO_URI="mongodb://localhost:37017/?replicaSet=rs0" \
	RELOQUENT_TEST_MONGO_DATABASE=reloquent_test \
	go test -v -count=1 -tags=integration ./test/integration/...
