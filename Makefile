.PHONY: all
all:
	docker-compose rm -fsv
	docker-compose up --build -d

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
	docker-compose up -d singlestore redpanda

.PHONY: simulator
simulator:
	docker-compose rm -fsv simulator-0 simulator-1
	docker-compose up --build -d simulator-0 simulator-1

.PHONY: down
down:
	docker-compose down -v