.PHONY: all
all:
	docker-compose rm -fsv
	docker-compose up -d

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
	docker-compose rm -fsv simulator
	docker-compose up -d simulator

.PHONY: down
down:
	docker-compose down -v