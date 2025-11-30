# University Web Crawler & Search API

A concurrent web crawler written in Go. It scrapes US university data from Wikipedia, stores it in PostgreSQL, and provides a REST API for searching.

## Requirements

- Go 1.25+
- Docker (for PostgreSQL)

## Getting Started

Follow these steps to run the project locally.

### 1. Configuration

Create a `.env` file in the project root directory:

```env
DB_USER=admin
DB_PASSWORD=secret
DB_NAME=schools_db
DB_HOST=localhost
DB_PORT=5432
```

### 2. Start Database

Start a PostgreSQL container using Docker:

```bash
docker run --name wiki-db \
  -e POSTGRES_USER=admin \
  -e POSTGRES_PASSWORD=secret \
  -e POSTGRES_DB=schools_db \
  -p 5432:5432 \
  -d postgres:15-alpine
```

### 3. Run the Application

This tool supports three modes. You should run the crawl mode first to populate the database.

#### Mode 1: Crawl Data

Scrapes data from Wikipedia and saves it to the database.

```bash
go run cmd/server/main.go -mode=crawl
```

#### Mode 2: Search CLI

Test the search function directly in the terminal.

```bash
go run cmd/server/main.go -mode=search -query="Stanford"
```

#### Mode 3: API Server

Starts the HTTP server on port 8080.

```bash
go run cmd/server/main.go -mode=server
```

Test the API:
Open your browser or use curl:
http://localhost:8080/schools?q=stanford

You may see the result

```json
[
  {
    "name": "Leland Stanford Junior University",
    "aliases": ["Stanford University"]
  }
]
```

## Project Structure

- cmd/server/: Main entry point.
- internal/crawler/: Logic for web scraping (Worker Pool).
- internal/storage/: Database interaction (PostgreSQL).
- internal/api/: HTTP server handlers.
- internal/model/: Data structures.
