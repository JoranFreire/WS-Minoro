.PHONY: up down build test migrate logs benchmark tidy

# Start all services
up:
	cd infra && docker compose up -d

# Stop all services
down:
	cd infra && docker compose down

# Build all Docker images
build:
	cd infra && docker compose build

# Run tests for all Go services
test:
	cd services/link-router && go test ./... -race -cover
	cd services/link-admin && go test ./... -race -cover
	cd services/analytics-worker && go test ./... -race -cover

# Run DB migrations only
migrate:
	cd infra && docker compose run --rm migrate

# View logs
logs:
	cd infra && docker compose logs -f

# Benchmark link-router (requires wrk)
benchmark:
	wrk -t12 -c400 -d30s http://localhost:8080/health

# Tidy all Go modules
tidy:
	cd services/link-router && go mod tidy
	cd services/link-admin && go mod tidy
	cd services/analytics-worker && go mod tidy

# Run dashboard in dev mode
dashboard:
	cd dashboard && npm run dev

# Initialize Cassandra schema
cassandra-init:
	docker exec -i wsminoro-cassandra cqlsh < infra/cassandra_schema.cql
