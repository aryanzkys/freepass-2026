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

Optional AI variables:

- GEMINI_API_KEY
- GEMINI_MODEL
- AI_TIMEOUT_SECONDS

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

## AI Feedback Analyzer

AI endpoints are read-only and use aggregate data only. User identifiers are not included in prompts or responses.

Model default: gemini-2.5-flash

If GEMINI_API_KEY is missing, AI endpoints return 503 ai_not_configured.

## AI examples (PowerShell)

Owner feedback insights

try {
	Invoke-RestMethod -Method Get -Uri "http://localhost:8080/owner/canteens/$env:CANTEEN_ID/ai/feedback-insights?days=30" -Headers @{Authorization="Bearer $env:OWNER_TOKEN"}
} catch {
	$_.Exception.Message
}

User recommendation explain

try {
	Invoke-RestMethod -Method Get -Uri "http://localhost:8080/canteens/$env:CANTEEN_ID/ai/recommendation-explain?days=30" -Headers @{Authorization="Bearer $env:TOKEN"}
} catch {
	$_.Exception.Message
}

User recommendation explain by menu item

try {
	Invoke-RestMethod -Method Get -Uri "http://localhost:8080/canteens/$env:CANTEEN_ID/ai/recommendation-explain?days=30&menu_item_id=$env:MENU_ID" -Headers @{Authorization="Bearer $env:TOKEN"}
} catch {
	$_.Exception.Message
}

Admin oversight

try {
	Invoke-RestMethod -Method Get -Uri "http://localhost:8080/admin/ai/oversight?days=30" -Headers @{Authorization="Bearer $env:ADMIN_TOKEN"}
} catch {
	$_.Exception.Message
}

## Payment methods

The API supports CASH and CASHLESS_QRIS.

CASH flow:

- order_status starts WAITING
- payment_status starts UNPAID

CASHLESS_QRIS flow:

- order_status starts PAYMENT
- payment_status starts UNPAID
- user confirms payment, order_status becomes WAITING and payment_status becomes AWAITING_VERIFICATION
- owner approves to set payment_status PAID or rejects to set payment_status REJECTED and order_status PAYMENT with a refund record

## QRIS setup

Admin uploads a static QRIS URL per canteen using:

PUT /admin/canteens/:canteenId/qris

## Payment examples (PowerShell)

Create CASH order

Invoke-RestMethod -Method Post -Uri "http://localhost:8080/orders" -Headers @{Authorization="Bearer $env:TOKEN"} -ContentType "application/json" -Body '{"canteen_id":"'$env:CANTEEN_ID'","payment_method":"CASH","items":[{"menu_item_id":"'$env:MENU_ID'","qty":1}]}'

Create CASHLESS_QRIS order

Invoke-RestMethod -Method Post -Uri "http://localhost:8080/orders" -Headers @{Authorization="Bearer $env:TOKEN"} -ContentType "application/json" -Body '{"canteen_id":"'$env:CANTEEN_ID'","payment_method":"CASHLESS_QRIS","items":[{"menu_item_id":"'$env:MENU_ID'","qty":1}]}'

Get QRIS info

Invoke-RestMethod -Method Get -Uri "http://localhost:8080/orders/$env:ORDER_ID/payment/qris" -Headers @{Authorization="Bearer $env:TOKEN"}

Confirm QRIS paid

Invoke-RestMethod -Method Post -Uri "http://localhost:8080/orders/$env:ORDER_ID/payment/qris/confirm" -Headers @{Authorization="Bearer $env:TOKEN"}

Owner approve QRIS

Invoke-RestMethod -Method Patch -Uri "http://localhost:8080/owner/canteens/$env:CANTEEN_ID/orders/$env:ORDER_ID/payment/verify" -Headers @{Authorization="Bearer $env:OWNER_TOKEN"} -ContentType "application/json" -Body '{"action":"APPROVE"}'

Owner reject QRIS

Invoke-RestMethod -Method Patch -Uri "http://localhost:8080/owner/canteens/$env:CANTEEN_ID/orders/$env:ORDER_ID/payment/verify" -Headers @{Authorization="Bearer $env:OWNER_TOKEN"} -ContentType "application/json" -Body '{"action":"REJECT","reason":"Amount mismatch"}'

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

