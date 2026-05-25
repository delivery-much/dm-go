# Copilot Instructions — dm-go

## Project Overview

> * [Overview](#overview)

- **Primary Language**: Go
- **Runtime**: Go
- **Frameworks**: Gin, Chi
- **Data Stores**: MongoDB
- **Messaging**: RabbitMQ
- **Organization**: delivery-much

## Project Structure

This project follows a **Middleware** pattern.

## Architecture Context

This service is part of the Delivery Much microservices ecosystem.
Services communicate via REST APIs and RabbitMQ for async messaging.
This service uses: MongoDB.
Async messaging via: RabbitMQ.
Use the MCP server (mcp-jarvis) tools to understand dependencies, callers, and event flows.

## Internal Packages

This project uses the following Delivery Much internal packages:
- `dm-go`
- `mock-helper`

## Code Conventions

- Follow standard Go project layout
- Handle errors explicitly — check every returned error
- Use `context.Context` for request-scoped data and cancellation
- Prefer table-driven tests
- Use `gofmt` and `golint` for formatting
- Use Gin context helpers for request/response handling
- Use Chi router middleware chain pattern

## Testing

- Run tests: `go test ./...`
- Write tests for new features and bug fixes
- Mock external service calls and database connections in tests

## Commit Conventions

Use [Conventional Commits](https://www.conventionalcommits.org/):
- `feat:` new features
- `fix:` bug fixes
- `docs:` documentation changes
- `chore:` maintenance tasks
- `test:` test additions/modifications
- `refactor:` code refactoring without behavior change

## Delivery Much Context

- **Databases**: MongoDB (primary), MariaDB (legacy), Redis (cache)
- **Messaging**: RabbitMQ (exchanges, queues, routing keys)
- **Deploy**: Docker on AWS EC2, organized in stacks
- **Auth**: Auth0 (JWT) for external APIs, internal service-to-service via restauth-eden
