.PHONY: all
all: down monitoring storage simulator

.PHONY: logs
logs:
	docker-compose logs --tail 100 -f $(SERVICE)

.PHONY: monitoring
monitoring:
	docker-compose rm -fsv prometheus grafana
	docker-compose up -d prometheus grafana

.PHONY: storage
storage:
	docker-compose rm -fsv singlestore redpanda
	docker-compose up -d redpanda
	sleep 2
	docker-compose up -d redpanda-setup
	docker-compose up -d singlestore

.PHONY: simulator
simulator:
	docker-compose rm -fsv simulator
	docker-compose up --build -d simulator

.PHONY: down
down:
	docker-compose down -v