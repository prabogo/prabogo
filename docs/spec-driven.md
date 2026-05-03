# Spec-Driven Development

Prabogo is integrated with [Spec Kit](https://github.com/github/spec-kit) to enable spec-driven development. Instead of traditional agent roles, you guide GitHub Copilot (or another supported AI agent) through a structured workflow using slash commands. This approach ensures clear requirements, proper planning, and precise implementation.

## The Spec-Driven Development Workflow

Follow these seven steps to develop features systematically:

### 1. Define Your Constitution

**Command**: `/speckit.constitution`

Establish the core rules and principles that guide your project. This is the foundation for all subsequent development.

```
/speckit.constitution This project follows a Library-First approach. All features are independently testable. We use TDD strictly. Code must be fully documented.
```

### 2. Define Requirements with the Specification

**Command**: `/speckit.specify`

Describe what you want to build—focus on the **what** and **why**, not the tech stack. Be specific about user needs and features.

```
/speckit.specify Build a client management system that allows users to create and track clients, assign client managers, and maintain client contact information and project history.
```

### 3. Refine the Specification

**Command**: `/speckit.clarify`

Identify and resolve ambiguities in your specification. This iterative step ensures clarity before moving to planning.

```
/speckit.clarify Clarify how client managers are assigned. Can one client have multiple managers? How are manager permissions handled?
```

You can run `/speckit.clarify` multiple times to resolve different aspects of your spec.

### 4. Validate the Specification

**Command**: `/speckit.checklist`

Generate a custom validation checklist to ensure your specification is complete, clear, and comprehensive.

```
/speckit.checklist
```

### 5. Create a Technical Implementation Plan

**Command**: `/speckit.plan`

Provide your tech stack and architecture choices. Be specific about technologies, databases, frameworks, and technical constraints.

```
/speckit.plan This uses the Go Prabogo framework with PostgreSQL for storage, RabbitMQ for async messaging, Fiber for HTTP endpoints, and a hexagonal architecture pattern.
```

### 6. Break Down into Actionable Tasks

**Command**: `/speckit.tasks`

Generate a detailed, dependency-ordered task list ready for implementation.

```
/speckit.tasks
```

### 7. Validate the Plan and Implement

**Commands**: `/speckit.analyze` and `/speckit.implement`

First, audit the implementation plan for consistency and completeness:

```
/speckit.analyze
```

Then execute all tasks to build your feature:

```
/speckit.implement
```

## Key Principles

- **Be explicit**: Clearly describe what you're building and why
- **Separate concerns**: Focus on requirements during specification, tech stack during planning
- **Iterate before coding**: Refine and validate specs before implementation begins
- **Validate early**: Use `/speckit.analyze` before `/speckit.implement` to catch issues
- **Phased implementation**: For large features, implement in phases to avoid context saturation

## Example Workflow

Here's a complete example of building a feature in Prabogo:

```
1. /speckit.constitution Prabogo follows hexagonal architecture. All features are independently testable. TDD is mandatory.

2. /speckit.specify Build a client upsert system that validates and persists client data to the database.

3. /speckit.clarify What validation rules apply to client data? How are conflicts handled?

4. /speckit.checklist

5. /speckit.plan Use PostgreSQL migrations for schema changes, RabbitMQ for event publishing, and domain-driven validation.

6. /speckit.tasks

7. /speckit.analyze
   /speckit.implement
```

## Learn More

- [Spec Kit Documentation](https://github.github.io/spec-kit/)
- [Complete Spec-Driven Development Methodology](https://github.com/github/spec-kit/blob/main/spec-driven.md)
- [Spec Kit Quick Start](https://github.github.com/spec-kit/quickstart.html)
