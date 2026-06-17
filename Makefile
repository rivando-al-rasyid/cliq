include ./.env

MIGRATION_PATH=database/migrations
SEED_FILE=database/seed.sql
DATABASE_URL=postgresql://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

# ── Migration ──────────────────────────────────────────────────────────────
migrate-create:
	@migrate create -ext sql -dir $(MIGRATION_PATH) -seq create_$(NAME)_table

migrate-up:
	@migrate -database $(DATABASE_URL) -path $(MIGRATION_PATH) up

migrate-down:
	@migrate -database $(DATABASE_URL) -path $(MIGRATION_PATH) down

migrate-force:
	@migrate -database $(DATABASE_URL) -path $(MIGRATION_PATH) force $(VERSION)

migrate-status:
	@migrate -database $(DATABASE_URL) -path $(MIGRATION_PATH) version

# ── Seed ───────────────────────────────────────────────────────────────────
seed:
	@PGPASSWORD=$(DB_PASS) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -f $(SEED_FILE)

seed-reset:
	@PGPASSWORD=$(DB_PASS) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -c \
		"TRUNCATE TABLE withdrawals, expenses, transfers, transactions, topups, wallets, user_pins, favorites, profiles, users RESTART IDENTITY CASCADE;"
	@$(MAKE) seed

docker-up: 
	@docker compose up -d --build
docker-down: 
	@docker compose down -v
# ── Misc ───────────────────────────────────────────────────────────────────
print-db-url:
	@echo $(DATABASE_URL)