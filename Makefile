BINARY := bin/kitchen
ENV_FILE := config/kitchen.env
COMPOSE := docker compose -f deploy/docker-compose.yml

.PHONY: build run fmt vet lint test clean docs diagrams up down logs

build:
	go build -o $(BINARY) ./services/kitchen/cmd/app

docs:
	swag init -g services/kitchen/cmd/app/main.go -o docs --parseDependency --parseInternal

diagrams:
	docker run --rm -v $(PWD)/docs/diagrams:/work -w /work plantuml/plantuml -tsvg *.puml

run: build
	set -a; . ./$(ENV_FILE); set +a; \
	DB_CONN_STRING=$$(echo "$$DB_CONN_STRING" | sed 's/@kitchen-db:/@localhost:/'); \
	REDIS_ADDR=$$(echo "$$REDIS_ADDR" | sed 's/^kitchen-redis:/localhost:/'); \
	export DB_CONN_STRING REDIS_ADDR; \
	./$(BINARY)

up:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down -v

logs:
	$(COMPOSE) logs -f

fmt:
	gofmt -l .

vet:
	go vet ./...

lint:
	golangci-lint run

test:
	go test ./tests/...

clean:
	rm -f $(BINARY)
