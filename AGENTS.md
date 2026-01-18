# Prabogo Project Instructions for AI Assistants

This file contains instructions for AI coding assistants working with the Prabogo project.

## Project Information

- **Project Name**: Prabogo
- **Author**: Moch Dieqy Dzulqaidar
- **License**: MIT License
- **Go Version**: >= go1.24.0

## Project Structure
Prabogo uses a hexagonal architecture (ports and adapters) with the following structure:
- `cmd/`: Application entry point
  - `main.go`: Main application entrypoint
- `internal/`: Internal implementations
  - `app.go`: Application initialization
  - `adapter/`: Adapters (inbound and outbound)
    - `inbound/`: Input adapters
      - `command/`: CLI command handlers
      - `fiber/`: HTTP server using Fiber framework
      - `rabbitmq/`: Message consumers for RabbitMQ
    - `outbound/`: Output adapters
      - `http/`: HTTP clients
      - `postgres/`: PostgreSQL database adapters
      - `rabbitmq/`: RabbitMQ message publishers
      - `redis/`: Redis cache adapters
  - `domain/`: Domain logic
  - `migration/`: Migration scripts
    - `postgres/`: PostgreSQL migrations
  - `model/`: Data models
  - `port/`: Port definitions
    - `inbound/`: Input port definitions
    - `outbound/`: Output port definitions
- `design-docs/`: Design documentation
  - `images/`: Documentation images/diagrams
- `tests/`: Test files and mocks
  - `mocks/`: Mock implementations for testing
- `utils/`: Utility functions
  - `activity/`: Activity tracking
  - `database/`: Database utilities
  - `google/`: Google service utilities
  - `log/`: Logging utilities
  - `rabbitmq/`: RabbitMQ utilities
  - `redis/`: Redis utilities

## Supporting Files
- `docker-compose.yml`: Docker Compose configuration
- `Dockerfile`: Docker image definition
- `go.mod` and `go.sum`: Go module definitions
- `LICENSE`: MIT license file
- `Makefile`: Build and development automation
- `README.md`: Project documentation

## Code Conventions
- Uses Go modules
- Follows Go naming standards (camelCase for private, PascalCase for public)
- Unit tests for all domain logic

## Technologies
- PostgreSQL for database
- RabbitMQ for message queue
- Redis for caching
- Fiber for HTTP server

## Development Patterns
- Use Makefile for common operations (see README.md for details)
- Use Docker for external services
- Hexagonal architecture for clean separation of concerns

## Code Preferences
- Avoid using global variables
- Use dependency injection
- Follow clean architecture principles
- Prioritize code readability
- Use interfaces for component interaction

## Important Notes
- Do not change the main project structure
- Always run unit tests after significant changes
- Document APIs and important functions
- Use Makefile targets for code generation

## Copyright Information
Copyright (c) 2025 Moch Dieqy Dzulqaidar
