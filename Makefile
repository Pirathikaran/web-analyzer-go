BINARY     := web-analyzer
PORT       ?= 8080
IMAGE      := web-analyzer:latest

.PHONY: build run test test-cover lint docker-build docker-run clean

## build: compile the server binary
build:
	go build -ldflags="-s -w" -o $(BINARY) ./cmd/server

## run: build and run locally
run: build
	PORT=$(PORT) ./$(BINARY)

## test: run all unit tests
test:
	go test ./... -race -count=1

## test-cover: run tests and show coverage report
test-cover:
	go test ./... -race -coverprofile=coverage.out
	go tool cover -func=coverage.out

## lint: vet and staticcheck
lint:
	go vet ./...

## docker-build: build the Docker image
docker-build:
	docker build -t $(IMAGE) .

## docker-run: run the Docker container
docker-run: docker-build
	docker run --rm -p $(PORT):8080 $(IMAGE)

## clean: remove build artifacts
clean:
	rm -f $(BINARY) coverage.out
