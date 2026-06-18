# Cliq Backend

Cliq Backend is a REST API for a shortlink / URL shortener application built with **Go**, **Gin**, **PostgreSQL**, and **Redis**.

It handles authentication, shortlink creation, custom slug management, link redirection, profile management, password reset, and protected user routes.

> Cliq is a learning and portfolio project.

---

## Tech Stack

* Go
* Gin
* PostgreSQL
* Redis
* JWT Authentication
* Docker
* Docker Compose
* golang-migrate
* Swagger

---

## Project Structure

```txt
cliq/
├── database/
│   ├── migrations/
│   └── seed.sql
├── docs/
├── internals/
│   ├── config/
│   ├── controller/
│   ├── dto/
│   ├── middleware/
│   ├── model/
│   ├── pkg/
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

---

## Requirements

### Local Development

* Go
* PostgreSQL
* Redis
* golang-migrate
* Make

### Docker Development

* Docker
* Docker Compose

---

## Environment Variables

Copy the example environment file:

```bash
cp env.example .env
```

Example `.env`:

```env
APP_PORT=8080

DB_USER=cliq
DB_PASS=secret
DB_NAME=cliq_db
DB_HOST=postgres
DB_PORT=5432

RDB_HOST=redis
RDB_PORT=6379
RDB_USER=
RDB_PASS=

JWT_SECRET=change_me_to_a_long_random_string
JWT_ISSUER=cliq
```

For local development without Docker:

```env
DB_HOST=localhost
RDB_HOST=localhost
```

Do not commit `.env` to Git.

---

## Run with Docker

From the project root:

```bash
docker compose up -d --build
```

This starts:

* Go backend API
* PostgreSQL
* Redis
* Migration service

Check running containers:

```bash
docker ps
```

Stop containers:

```bash
docker compose down
```

Stop containers and remove volumes:

```bash
docker compose down -v
```

---

## Run Locally

Install dependencies:

```bash
go mod download
```

Make sure PostgreSQL and Redis are running.

Run database migrations:

```bash
make migrate-up
```

Run seed data if needed:

```bash
make seed
```

Start the backend:

```bash
go run .
```

If your entry point is inside `cmd`, use:

```bash
go run ./cmd
```

The backend runs at:

```txt
http://localhost:8080
```

---

## API Base URL

Direct backend access:

```txt
http://localhost:8080
```

Frontend access through Nginx proxy:

```txt
/api
```

Recommended frontend environment variable:

```env
VITE_API_BASE_URL=/api
```

---

## Authentication

Protected endpoints require a Bearer token:

```txt
Authorization: Bearer <token>
```

Example:

```bash
curl "http://localhost:8080/profile" \
  -H "Authorization: Bearer <token>"
```

---

## API Routes

### Auth Routes

```txt
POST /auth/register
POST /auth/login
POST /auth/reset
POST /auth/reset/confirm
POST /auth/change-password
POST /auth/logout
```

### Profile Routes

```txt
GET   /profile
GET   /profile/info
PATCH /profile/edit
PATCH /profile/change/password
```

### Shortlink Routes

```txt
POST /cliq
GET  /cliq
GET  /cliq/:id
PATCH /cliq/:id
DELETE /cliq/:id
```

### Redirect Route

```txt
GET /:slug
```

The redirect route is used to open the original URL from a shortlink slug.

Example:

```txt
GET /github
```

Redirects to the original link saved for the `github` slug.

---

## Shortlink Feature

Cliq allows authenticated users to create shortlinks.

A shortlink usually contains:

* original URL
* custom slug
* generated short URL
* owner user ID
* created date
* updated date

Example request:

```bash
curl -X POST "http://localhost:8080/cliq" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "origin_link": "https://github.com/rivando-al-rasyid",
    "slug": "github"
  }'
```

Example response:

```json
{
  "message": "shortlink created successfully",
  "data": {
    "slug": "github",
    "short_url": "http://localhost:8080/github",
    "origin_link": "https://github.com/rivando-al-rasyid"
  },
  "isSuccess": true
}
```

---

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

Only force a migration version when you are sure about the current database state.

---

## Seed Database

Run seed:

```bash
make seed
```

Reset seed:

```bash
make seed-reset
```

The reset command truncates core tables and inserts seed data again.

Use this carefully because it removes existing development data.

---

## Swagger Documentation

Swagger documentation is available at:

```txt
GET /swagger/index.html
```

Example:

```txt
http://localhost:8080/swagger/index.html
```

---

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

---

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

---

## API Design Notes

* `POST /cliq` is used to create a new shortlink.
* `GET /:slug` is used to redirect users to the original URL.
* Slugs should be unique.
* Protected routes should require JWT authentication.
* Public redirect routes should not require authentication.
* Keep sensitive values inside `.env`.
* Do not commit `.env` to Git.

---

## Security Notice

Cliq is a learning and portfolio project.

Before production use, review at minimum:

* Authentication and JWT security
* Password hashing
* Password reset token security
* Slug validation
* URL validation
* Rate limiting
* Abuse prevention
* Redirect safety
* Open redirect protection
* Input validation
* CORS policy
* Logging and monitoring
* Secret management
* Backup and recovery

---

## Common Issues

### Database Connection Failed

Check:

```env
DB_HOST
DB_PORT
DB_USER
DB_PASS
DB_NAME
```

If using Docker:

```env
DB_HOST=postgres
```

If running locally:

```env
DB_HOST=localhost
```

---

### Redis Connection Failed

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

---

### Migration Dirty Error

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

---

## License

This project is licensed under the MIT License.

See the `LICENSE` file for details.
