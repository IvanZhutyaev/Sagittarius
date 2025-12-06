.PHONY: build run test clean docker-build docker-up docker-down migrate

# Build all services
build:
	@echo "Building all services..."
	@cd services/api-gateway && go build -o ../../bin/api-gateway ./cmd/main.go
	@cd services/auction-engine && go build -o ../../bin/auction-engine ./cmd/main.go
	@cd services/bidder && go build -o ../../bin/bidder ./cmd/main.go
	@cd services/budget && go build -o ../../bin/budget ./cmd/main.go
	@cd services/notification && go build -o ../../bin/notification ./cmd/main.go

# Run all services locally
run:
	@echo "Starting all services..."
	@docker-compose up -d

# Run tests
test:
	@echo "Running tests..."
	@go test ./... -v -cover

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -rf *.log

# Docker operations
docker-build:
	@echo "Building Docker images..."
	@docker-compose build

docker-up:
	@echo "Starting Docker containers..."
	@docker-compose up -d

docker-down:
	@echo "Stopping Docker containers..."
	@docker-compose down

# Database migrations
migrate:
	@echo "Running migrations..."
	@cd migrations && go run main.go

