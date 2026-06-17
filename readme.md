# VanWallet Backend

VanWallet Backend is a REST API built with **Go**, **Gin**, **PostgreSQL**, and **Redis**. It handles authentication, profile management, wallet transactions, top-up, transfer, withdrawal, expense tracking, transaction history, receiver search, dashboard summary, and reports.

## Tech Stack

* Go `1.26.3`
* Gin
* PostgreSQL
* Redis
* JWT Authentication
* Docker
* Docker Compose
* golang-migrate
* Swagger

## Go Version

This project uses:

```txt
go 1.26.3
```

Make sure your local machine, Docker image, and CI environment use Go `1.26.3`.

Check your installed Go version:

```bash
go version
```

Expected result:

```txt
go version go1.26.3 linux/amd64
```

If you use Docker, make sure the backend `Dockerfile` uses a matching Go image:

```dockerfile
FROM golang:1.26.3-alpine AS builder
```

## Project Structure

```txt
backend/
├── database/
│   ├── migrations/
│   └── seed.sql
├── docs/
├── internals/
│   ├── controller/
│   ├── middleware/
│   ├── model/
│   ├── repository/
│   ├── router/
│   └── service/
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── go.mod
├── go.sum
└── env.example
```

## Requirements

For local development:

* Go `1.26.3`
* PostgreSQL
* Redis
* golang-migrate
* Make

For Docker development:

* Docker
* Docker Compose

## Environment Variables

Copy the example environment file:

```bash
cp env.example .env
```

Example `.env`:

```env
DB_USER=vanwallet
DB_PASS=secret
DB_NAME=vanwallet_db
DB_HOST=postgres
DB_PORT=5432

RDB_HOST=redis
RDB_PORT=6379
RDB_USER=
RDB_PASS=

JWT_SECRET=change_me_to_a_long_random_string
JWT_ISSUER=vanwallet
```

For local development without Docker:

```env
DB_HOST=localhost
DB_PORT=5432

RDB_HOST=localhost
RDB_PORT=6379
```

## Run with Docker

From the backend folder:

```bash
docker compose up -d --build
```

This starts the backend services, including:

* Go backend API
* PostgreSQL
* Redis
* Migration service

Stop all containers:

```bash
docker compose down
```

Stop and remove volumes:

```bash
docker compose down -v
```

## Run Locally

Install Go dependencies:

```bash
go mod download
```

Make sure PostgreSQL and Redis are running.

Run database migration:

```bash
make migrate-up
```

Run seed data if needed:

```bash
make seed
```

Start the backend:

```bash
go run main.go
```

The backend runs on:

```txt
http://localhost:8080
```

## Database Migration

Create a new migration:

```bash
make migrate-create NAME=users
```

Run migrations:

```bash
make migrate-up
```

Rollback migrations:

```bash
make migrate-down
```

Check migration version:

```bash
make migrate-status
```

Force migration version:

```bash
make migrate-force VERSION=1
```

## Seed Database

Run seed:

```bash
make seed
```

Reset seed:

```bash
make seed-reset
```

The seed reset command truncates core tables and inserts seed data again.

## API Base URL

If accessed directly:

```txt
http://localhost:8080
```

If accessed through frontend Nginx proxy:

```txt
/api
```

Recommended frontend API base:

```env
VITE_API_BASE_URL=/api
```

## Authentication

Protected endpoints require a Bearer token:

```txt
Authorization: Bearer <token>
```

## Auth Routes

```txt
POST /auth/register
POST /auth/login
POST /auth/reset
POST /auth/reset/confirm
POST /auth/change-password
POST /auth/logout
GET  /auth/pin
POST /auth/pin/verify
```

## Profile Routes

```txt
GET   /profile
GET   /profile/info
PATCH /profile/edit
PATCH /profile/change/pin
PATCH /profile/change/password
```

## Transaction Routes

```txt
GET   /transaction/summary
GET   /transaction/report
GET   /transaction/history
GET   /transaction/receiver

POST  /transaction/topup
PATCH /transaction/topup/:id/confirm
POST  /transaction/transfer
POST  /transaction/withdrawal
POST  /transaction/expense
```

## Transaction History

Main endpoint:

```txt
GET /transaction/history
```

This endpoint is the main transaction feed for the frontend history page and dashboard graph.

Supported query parameters:

```txt
page
limit
q
source
type
status
direction
start_date
end_date
wallet_id
```

Example request:

```bash
curl "http://localhost:8080/transaction/history?page=1&limit=10&q=bca&status=SUCCESS" \
  -H "Authorization: Bearer <token>"
```

Example response:

```json
{
  "data": [],
  "total": 25,
  "page": 1,
  "limit": 10,
  "total_pages": 3
}
```

### History Filter Examples

Get first page:

```txt
GET /transaction/history?page=1&limit=10
```

Search transaction:

```txt
GET /transaction/history?page=1&limit=10&q=bca
```

Filter by status:

```txt
GET /transaction/history?page=1&limit=10&status=SUCCESS
```

Filter by direction:

```txt
GET /transaction/history?page=1&limit=10&direction=income
GET /transaction/history?page=1&limit=10&direction=expense
```

Filter by transaction type:

```txt
GET /transaction/history?page=1&limit=10&type=TOPUP
GET /transaction/history?page=1&limit=10&type=TRANSFER_IN
GET /transaction/history?page=1&limit=10&type=TRANSFER_OUT
GET /transaction/history?page=1&limit=10&type=WITHDRAWAL
GET /transaction/history?page=1&limit=10&type=EXPENSE
```

Filter by date range:

```txt
GET /transaction/history?page=1&limit=10&start_date=2026-06-01&end_date=2026-06-30
```

## Receiver Search

Main endpoint:

```txt
GET /transaction/receiver
```

This endpoint is used by the transfer page to show all available receiver users.

Supported query parameters:

```txt
page
limit
q
query
```

Example request:

```bash
curl "http://localhost:8080/transaction/receiver?page=1&limit=10&q=andi" \
  -H "Authorization: Bearer <token>"
```

Example response:

```json
{
  "data": [],
  "total": 30,
  "page": 1,
  "limit": 10,
  "total_pages": 3
}
```

Search supports:

```txt
full_name
email
phone
wallet_label
wallet_id
```

### Receiver Examples

Show all receiver users:

```txt
GET /transaction/receiver?page=1&limit=10
```

Search receiver:

```txt
GET /transaction/receiver?page=1&limit=10&q=andi
```

Alternative search query:

```txt
GET /transaction/receiver?page=1&limit=10&query=andi
```

## Transaction Actions

### Top Up

```txt
POST /transaction/topup
```

Confirm top-up:

```txt
PATCH /transaction/topup/:id/confirm
```

### Transfer

```txt
POST /transaction/transfer
```

### Withdrawal

```txt
POST /transaction/withdrawal
```

### Expense

```txt
POST /transaction/expense
```

## Dashboard Summary

```txt
GET /transaction/summary
```

Used by the dashboard to show:

* current balance
* total income
* total expense
* wallet summary

## Transaction Report

```txt
GET /transaction/report
```

Used for financial report data.

The frontend dashboard chart mainly uses transaction history data so the graph can support:

* 7 days
* 14 days
* 30 days
* income filter
* expense filter
* all filter

## Swagger

Swagger documentation is available at:

```txt
GET /swagger/index.html
```

Example:

```txt
http://localhost:8080/swagger/index.html
```

## Useful Make Commands

```bash
make migrate-create NAME=table_name
make migrate-up
make migrate-down
make migrate-status
make migrate-force VERSION=1
make seed
make seed-reset
make print-db-url
```

## Recommended Development Flow

Start services:

```bash
docker compose up -d --build
```

Check containers:

```bash
docker ps
```

Run migrations manually if needed:

```bash
make migrate-up
```

Run seed manually if needed:

```bash
make seed
```

Run tests:

```bash
go test ./...
```

## API Design Notes

* `GET /transaction/history` is the main transaction list endpoint.
* `GET /transaction/receiver` is the main receiver/user list endpoint for transfer.
* Transaction creation endpoints should stay separate from transaction reading endpoints.
* Avoid re-adding duplicate transaction list/detail routes unless they are really needed.
* Keep pagination and filtering on list endpoints.
* Keep sensitive values inside `.env`.
* Do not commit `.env` to Git.

## Security Notice

VanWallet is a learning and portfolio project. It should not be used in production or for real financial transactions without a full security audit, compliance review, infrastructure hardening, monitoring, and proper financial/legal approval.

Before production use, review at minimum:

* authentication and JWT security
* password hashing
* PIN handling
* transaction validation
* race condition protection
* balance consistency
* database transaction safety
* input validation
* rate limiting
* logging and monitoring
* CORS policy
* secret management
* backup and recovery
* financial compliance requirements

## Common Issues

### Database connection failed

Check:

```env
DB_HOST
DB_PORT
DB_USER
DB_PASS
DB_NAME
```

If using Docker, the database host should usually be the Docker service name:

```env
DB_HOST=postgres
```

If running locally:

```env
DB_HOST=localhost
```

### Redis connection failed

Check:

```env
RDB_HOST
RDB_PORT
RDB_USER
RDB_PASS
```

If using Docker:

```env
RDB_HOST=redis
```

If running locally:

```env
RDB_HOST=localhost
```

### Migration dirty error

Check migration status:

```bash
make migrate-status
```

Force the correct version only if you are sure:

```bash
make migrate-force VERSION=1
```

Then run migration again:

```bash
make migrate-up
```

### Go version error

This project uses Go `1.26.3`.

Check your version:

```bash
go version
```

If the version is different, install Go `1.26.3` or use a matching Docker image.

## License

This project is licensed under the MIT License.

See the `LICENSE` file for details.
