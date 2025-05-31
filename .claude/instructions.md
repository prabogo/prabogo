# Claude Instructions for Prabogo Project

This file contains specific instructions for Claude AI when working with the Prabogo project.

## Project Information

- **Project Name**: Prabogo
- **Author**: Moch Dieqy Dzulqaidar
- **License**: MIT License
- **Go Version**: >= go1.23.0

## Project Overview
Prabogo is a Go application using hexagonal architecture that separates business logic from external concerns. The codebase is organized with clean separation between domain logic and adapters that connect to external systems.

## Coding Guidelines for Claude

### Architecture
- Respect the hexagonal architecture pattern used in this project
- Keep domain logic independent of external frameworks and libraries
- Use adapter pattern for all external dependencies

### Code Style
- Follow Go standard naming conventions (camelCase for private, PascalCase for public)
- Implement comprehensive error handling
- Document public interfaces and complex logic
- Write unit tests for all domain logic

### Development Workflow
- Use provided Makefile commands for code generation and other tasks
- Always run tests after significant changes
- Consult the README.md for detailed operational instructions

## Directory Structure
See AI_INSTRUCTIONS.md or .github/copilot/instructions.md for the complete directory structure.

## Copyright Information
Copyright (c) 2025 Moch Dieqy Dzulqaidar
