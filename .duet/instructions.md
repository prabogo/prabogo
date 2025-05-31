# Duet AI Instructions for Prabogo Project

This file contains specific instructions for Duet AI when working with the Prabogo project.

## Project Information

- **Project Name**: Prabogo
- **Author**: Moch Dieqy Dzulqaidar
- **License**: MIT License
- **Go Version**: >= go1.23.0

## Project Overview
Prabogo is a Go application that follows hexagonal architecture principles, separating domain logic from adapters. The project uses PostgreSQL for data storage, RabbitMQ for messaging, Redis for caching, and Fiber for HTTP handling.

## Coding Guidelines for Duet AI

### Architecture
- Follow the existing hexagonal (ports and adapters) architecture
- Domain logic should be pure and not depend on external frameworks
- Adapters should implement the interfaces defined in ports

### Code Style
- Follow Go standard naming conventions
- Use dependency injection for all components
- Write comprehensive unit tests for domain logic
- Document all public interfaces and complex functions

### Development Workflow
- Use Makefile commands for common tasks (see README.md)
- Run unit tests after making significant changes
- Follow the established project structure

## Directory Structure
See AI_INSTRUCTIONS.md or .github/copilot/instructions.md for the complete directory structure.

## Copyright Information
Copyright (c) 2025 Moch Dieqy Dzulqaidar
