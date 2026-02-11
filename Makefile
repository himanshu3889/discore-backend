# Make sure install migrate cli tool: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Postgres environment variables
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=discore
POSTGRES_USER=discore
POSTGRES_PASSWORD=discore


# Migrate commands

MIGRATE = migrate -path internal/modules/core/migrations -database "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable"

# (not real files)
.PHONY: migrate-up migrate-down migrate-version migrate-new

migrate-up:
	$(MIGRATE) up

migrate-down:
	$(MIGRATE) down 1

migrate-down-force:
	$(MIGRATE) force 1
	$(MIGRATE) down 1

migrate-version:
	$(MIGRATE) version

migrate-new:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir internal/modules/core/migrations -seq $${name}


# Docker project commands

up-db:
	docker compose up -d postgres mongodb redis

down-db:
	docker compose down postgres mongodb redis

up-mon:
	docker compose --profile monitoring up -d prometheus grafana

down-mon:
	docker compose down prometheus grafana

up-sys:
	docker compose up -d postgres mongodb redis kafka

down-sys:
	docker compose down


cont-urls:
	@echo "Prometheus: http://localhost:9090"
	@echo "Grafana: http://localhost:9300"




# KAKFA Commands
KAFKA_CONTAINER ?= discore-kafka
BOOTSTRAP_SERVER ?= localhost:9092

KAFKA_GROUP_CMD = docker exec -it $(KAFKA_CONTAINER) \
	/opt/kafka/bin/kafka-consumer-groups.sh \
	--bootstrap-server $(BOOTSTRAP_SERVER)

KAFKA_CONSUMER_CMD = docker exec -it $(KAFKA_CONTAINER) \
	/opt/kafka/bin/kafka-console-consumer.sh \
	--bootstrap-server $(BOOTSTRAP_SERVER)

KAFKA_TOPIC_CMD = docker exec -it $(KAFKA_CONTAINER) \
	/opt/kafka/bin/kafka-topics.sh \
	--bootstrap-server $(BOOTSTRAP_SERVER)

kafka-grp:
	$(KAFKA_GROUP_CMD) $(args)

kafka-grp-describe:
	$(KAFKA_GROUP_CMD) --describe --group $(group)


	