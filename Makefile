.DEFAULT_GOAL := help
DB ?= sqlite
COMPOSE = docker compose -f docker-compose.yml
ifeq ($(DB),postgres)
COMPOSE += -f docker/compose.postgres.yml
else ifneq ($(DB),sqlite)
$(error DB must be sqlite or postgres)
endif

.PHONY: help setup install up down logs ps migrate seed-api native-api native-worker native-migrate native-seed dev-web dev-mobile desktop test-go test-web test-mobile e2e-install e2e
help:
	@echo 'make setup                  generate a private .env (Python 3)'
	@echo 'make up [DB=postgres]       build and start the local stack (SQLite by default)'
	@echo 'make seed-api [DB=postgres] provision a real school admin (interactive or via EDUPILOT_SEED_*)'
	@echo 'make ps|logs|down [DB=postgres]    inspect or stop the same stack; volumes are retained'
	@echo 'make native-migrate native-api native-worker   native Go commands (run servers separately)'
	@echo 'make install                install client workspaces'
	@echo 'make dev-web|dev-mobile|desktop                 start or package a client'
	@echo 'make test-go|test-web|test-mobile|e2e            explicit test runs only'
setup:
	python3 scripts/setup.py
install:
	npm install
up: setup
	$(COMPOSE) up -d --build
down:
	$(COMPOSE) down --remove-orphans
logs:
	$(COMPOSE) logs --tail 100 -f
ps:
	$(COMPOSE) ps -a
migrate: setup
	$(COMPOSE) run --rm migrate
seed-api: setup
	bash scripts/seed-admin.sh $(DB)
native-migrate: setup
	go run ./cmd/migrate
native-api: setup
	go run .
native-worker: setup
	go run ./cmd/worker
native-seed: setup
	bash scripts/seed-admin.sh --native
dev-web:
	npm run dev --workspace @edupilot/web
dev-mobile:
	npm run start --workspace @edupilot/mobile
desktop:
	npm run build --workspace @edupilot/web
	node clients/desktop/scripts/build.mjs
test-go:
	go test ./...
test-web:
	npm run test --workspace @edupilot/web
test-mobile:
	npm run typecheck --workspace @edupilot/mobile
	npm run test --workspace @edupilot/mobile
e2e-install:
	cd tests/e2e && npm install --workspaces=false
	cd tests/e2e && npm exec --workspaces=false -- playwright install chromium
e2e:
	cd tests/e2e && npm run test --workspaces=false
