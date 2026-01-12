.PHONY: build run test bench docker-build clean fmt vet lint

BINARY_NAME=redis-clone
DOCKER_IMAGE=redis-clone:latest

build:
	go build -o $(BINARY_NAME) cmd/main.go

run: build
	./$(BINARY_NAME)

test:
	go test -v ./...

bench:
	go test -bench=. -benchmem ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	# Assuming golangci-lint is installed
	golangci-lint run

docker-build:
	docker build -t $(DOCKER_IMAGE) .

clean:
	go clean
	rm -f $(BINARY_NAME)
