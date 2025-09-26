.PHONY: help up down test bench clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

up: ## Start PostgreSQL in Docker
	docker-compose up -d
	@echo "⏳ Waiting for PostgreSQL to be ready..."
	@sleep 5
	@echo "✅ PostgreSQL is ready"

down: ## Stop PostgreSQL
	docker-compose down -v

test: ## Run performance test with real DB
	go run main.go

bench: ## Run Go benchmarks
	go test -bench=. -benchmem -benchtime=10s

build: ## Build all Go files
	go build main.go

clean: ## Clean up
	docker-compose down -v
	rm -f main

install: ## Install Go dependencies
	go mod init audience-poc 2>/dev/null || true
	go get github.com/lib/pq
	go get github.com/RoaringBitmap/roaring

demo: up ## Full demo: start DB and run test
	@sleep 5
	go run main.go

compare: ## Compare in-memory vs DB performance
	@echo "🔬 In-Memory Performance:"
	@go run poc.go | grep -E "(Старий|Bitmap|Прискорення)"
	@echo "\n🗄️  Database Performance:"
	@go run main.go | grep -E "(EAV|Optimized|Speedup)"