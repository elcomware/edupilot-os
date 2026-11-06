.DEFAULT_GOAL := help
DB ?= sqlite
COMPOSE = docker compose -f docker-compose.yml
ifeq ($(DB),postgres)
COMPOSE += -f docker/compose.postgres.yml
else ifneq ($(DB),sqlite)
$(error DB must be sqlite or postgres)
endif

.PHONY: help setup install up down logs ps migrate seed-api native-api native-worker native-migrate native-seed check-native docker-up docker-down docker-logs docker-ps docker-migrate docker-seed dev-web dev-mobile desktop test-go test-web test-mobile test-native e2e-install e2e
help:
	@echo 'make install                install client workspaces (Node matching package.json engines)'
	@echo 'make up                     migrate, build and supervise native API, worker and web (foreground)'
	@echo 'make seed-api               provision a school admin in another terminal (hidden password prompt)'
	@echo 'make check-native           read-only prerequisite check; Go 1.25+, Node, npm, Python 3, Make'
	@echo 'make down                   explain native stop; Ctrl-C in the make up terminal stops its processes'
	@echo 'make migrate                apply native database migrations from .env (SQLite default)'
	@echo 'make setup                  generate a private .env (also runs automatically at startup)'
	@echo 'make docker-up|docker-down|docker-logs|docker-ps [DB=postgres]   optional Compose stack'
	@echo 'make docker-migrate|docker-seed [DB=postgres]                 optional Compose operations'
	@echo 'make native-api|native-worker   run one native service separately'
	@echo 'make dev-web|dev-mobile|desktop  start or package a client'
	@echo 'make test-native|test-go|test-web|test-mobile|e2e              explicit checks'
setup:
	python3 scripts/setup.py
install:
	npm install
up:
	python3 scripts/dev.py
check-native:
	python3 scripts/dev.py --check
down:
	@echo 'Native services run in the foreground. Press Ctrl-C in the make up terminal to stop them.'
logs:
	@echo 'Native logs stream in the make up terminal. Use make docker-logs for Compose.'
ps:
	@echo 'Native services belong to the make up foreground supervisor. Use make docker-ps for Compose.'
migrate: setup
	go run ./cmd/migrate
seed-api: setup
	bash scripts/seed-admin.sh --native
native-migrate: migrate
native-api: setup
	go run .
native-worker: setup
	go run ./cmd/worker
native-seed: seed-api
docker-up: setup
	$(COMPOSE) up -d --build
docker-down:
	$(COMPOSE) down --remove-orphans
docker-logs:
	$(COMPOSE) logs --tail 100 -f
docker-ps:
	$(COMPOSE) ps -a
docker-migrate: setup
	$(COMPOSE) run --rm migrate
docker-seed: setup
	bash scripts/seed-admin.sh --docker $(DB)
dev-web:
	node node_modules/vite/bin/vite.js clients/web --config clients/web/vite.config.ts --host 127.0.0.1 --port 5173 --strictPort
dev-mobile:
	npm run start --workspace @edupilot/mobile
desktop:
	npm run build --workspace @edupilot/web
	node clients/desktop/scripts/build.mjs
test-native:
	python3 -m unittest discover -s scripts/tests -p 'test_dev.py' -v
test-go:
	go test ./...
test-web:
	npm run test --workspace @edupilot/web
test-mobile:
	npm run typecheck --workspace @edupilot/mobile
	npm run test --workspace @edupilot/mobile
e2e-install:
	cd tests/e2e && npm ci --workspaces=false
	cd tests/e2e && npm exec --workspaces=false -- playwright install chromium
e2e:
	cd tests/e2e && npm run test --workspaces=false

.PHONY: test-desktop
test-desktop:
	cd clients/desktop && go test ./internal/gateway
