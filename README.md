# BCC Canteen

BCC Canteen backend API built with Go, Gin, PostgreSQL, and sqlc.

## Tech stack

- Go
- Gin
- PostgreSQL
- sqlc

## Requirements

- Go 1.22+
- PostgreSQL 15+

## Environment

Copy the environment file and update values as needed.

cp .env.example .env

Required variables:

- DATABASE_URL
- JWT_SECRET
- APP_PORT

## Database setup

Option A: Local PostgreSQL

1. Create a database
2. Set DATABASE_URL in .env
3. Apply migrations

Option B: Docker Compose PostgreSQL

docker compose up -d

Set DATABASE_URL to:

postgres://postgres:postgres@localhost:5432/bcc_canteen?sslmode=disable

Option C: Supabase PostgreSQL

1. Create a Supabase project
2. Get the connection string
3. Set DATABASE_URL

## Migrations

If you do not use a migration tool, run the SQL files manually in order:

psql "$env:DATABASE_URL" -f migrations/000001_init.sql
psql "$env:DATABASE_URL" -f migrations/000002_improvements.sql
psql "$env:DATABASE_URL" -f migrations/000002_menu_ratings.sql

## sqlc

sqlc generate

## Run the server

go run ./cmd/api

## Run tests

go test ./...

## API documentation

Open docs/openapi.yaml in your preferred OpenAPI viewer.

## Admin testing

Register a normal user, then promote it to ADMIN using SQL:

UPDATE users SET role = 'ADMIN' WHERE email = 'admin@example.com';

## Example API calls (PowerShell)

Register

Invoke-RestMethod -Method Post -Uri "http://localhost:8080/auth/register" -ContentType "application/json" -Body '{"name":"Admin","email":"admin@example.com","password":"password123"}'

Login

Invoke-RestMethod -Method Post -Uri "http://localhost:8080/auth/login" -ContentType "application/json" -Body '{"email":"admin@example.com","password":"password123"}'

Create owner (ADMIN token required)

Invoke-RestMethod -Method Post -Uri "http://localhost:8080/admin/owners" -Headers @{Authorization="Bearer $env:TOKEN"} -ContentType "application/json" -Body '{"name":"Owner One","email":"owner@example.com","password":"password123","phone":"08123456789"}'

Update owner

Invoke-RestMethod -Method Put -Uri "http://localhost:8080/admin/owners/$env:OWNER_ID" -Headers @{Authorization="Bearer $env:TOKEN"} -ContentType "application/json" -Body '{"name":"Owner Updated","phone":"0899999999"}'

Delete account

Invoke-RestMethod -Method Delete -Uri "http://localhost:8080/admin/accounts/$env:USER_ID" -Headers @{Authorization="Bearer $env:TOKEN"}

