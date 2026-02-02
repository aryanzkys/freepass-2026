# BCC Canteen

Go backend service for BCC Canteen using Gin and PostgreSQL.

## Local setup

1. Install Go 1.22+
2. Copy environment file

	cp .env.example .env

3. Start PostgreSQL with Docker Compose

	docker compose up -d

4. Run the API

	make run

## Health check

curl http://localhost:8080/health

