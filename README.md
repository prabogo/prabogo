# Prabogo

![Alt text](./docs/images/icon.png "Prabogo Icon")

**Prabogo** is a Go framework designed to simplify project development by providing an interactive command interface and built-in instructions for AI assistance. This framework streamlines common engineering tasks, making it easier for software engineers to scaffold, generate, and manage project components efficiently. With Prabogo, developers benefit from automation and intelligent guidance, accelerating the software development process.

Prabogo now uses [Spec Kit](https://github.com/github/spec-kit) to bring spec-driven development into the project, making it easier to guide your AI agent with structured commands during development.

## Design Docs

[Design Docs List](./docs)

## Spec-Driven Development

[Learn how to use Spec Kit commands](./docs/spec-driven.md) to systematically develop features with your AI agent.

## Requirement

1. go version >= go1.24.0

## Spec Kit

This repository is initialized with [Spec Kit](https://github.com/github/spec-kit) for spec-driven development. With Spec Kit, Prabogo follows a spec-driven development workflow so developers can direct AI agents more clearly through structured planning and implementation commands.

GitHub Copilot is the default integration configured in this repository today, but Spec Kit also supports other AI coding agents. If you use a different agent, you can switch the integration to match your setup.

### Install Specify CLI

Spec Kit requires Python 3.11+ and assumes `uv` is already installed.

```sh
uv tool install specify-cli --from git+https://github.com/github/spec-kit.git@v0.8.4
specify version
```

### Initialize Spec Kit In This Repository

If you need to reinstall or refresh the Spec Kit files in this project, run:

```sh
specify init . --integration copilot
```

This installs the Spec Kit project files under `.specify/` and the GitHub Copilot integration files under `.github/agents/`, `.github/prompts/`, and `.github/copilot-instructions.md`.

### Use Another AI Agent

To see which integrations are available in your installed Spec Kit version, run:

```sh
specify integration list
```

If you use a different AI agent, reinitialize Spec Kit with the integration that matches your tool. For example:

```sh
specify init . --integration claude
```

Or:

```sh
specify init . --integration gemini
```

When you change the integration, Spec Kit rewrites the agent-specific command and instruction files for that tool. The core workflow stays the same: define principles, write specs, make a plan, generate tasks, and implement with structured agent commands.

### Main Spec Kit Commands

After initialization, you can use the generated Spec Kit commands in your configured AI agent:

```text
/speckit.constitution
/speckit.specify
/speckit.plan
/speckit.tasks
/speckit.implement
```

## Entire.io Integration

[Entire.io](https://docs.entire.io/cli/overview) captures context from AI agent-assisted code changes. For installation and detailed usage, see the [Entire CLI installation guide](https://docs.entire.io/cli/installation).

### Why Prabogo Recommends Entire.io

Prabogo uses Spec Kit for spec-driven development with AI agents. When combined with Entire.io, you get:
- **Full development context**: Capture prompts, decisions, and iterations alongside your code
- **Better collaboration**: Future team members understand not just what changed, but why and how it was developed
- **Session continuity**: Resume interrupted agent sessions instead of starting over
- **Clean history**: Checkpoint data lives on a separate Git branch, keeping your main history clean

**Quick start:**

```sh
entire enable
```

When working with AI agents, Entire automatically captures your session context and links it to commits without cluttering your Git history.

### View Checkpoints With Entire

To see the available captured sessions and checkpoints, run:

```sh
entire sessions list
```

To open the checkpoint explanation view for a specific checkpoint, use `entire explain -c <checkpoint-id>`:

```sh
entire explain -c <checkpoint-id>
```

Replace `<checkpoint-id>` with the checkpoint identifier shown in `entire sessions list`. When you are finished viewing the explanation, press `Ctrl+Z` to close it and return to your shell.

**Before running the app, copy the example environment file:**

```sh
cp .env.example .env
cp .env.docker.example .env.docker
```

## Start External Services with Docker Compose

```sh
docker-compose --env-file .env.docker up -d
```

## Stop External Services with Docker Compose

```sh
docker-compose down
```

## Start Authentik Services with Docker Compose

To start Authentik authentication services (includes PostgreSQL, Redis, Server, and Worker):

```sh
docker-compose -f docker-compose.authentik.yml up -d
```

## Stop Authentik Services with Docker Compose

```sh
docker-compose -f docker-compose.authentik.yml down
```

## Start Temporal Services with Docker Compose

To start Temporal workflow services (includes PostgreSQL, Elasticsearch, Temporal Server, Admin Tools, and Web UI):

```sh
docker-compose -f docker-compose.temporal.yml up -d
```

The Temporal UI will be available at http://localhost:8080 and the Temporal server will be accessible on port 7233.

## Stop Temporal Services with Docker Compose

```sh
docker-compose -f docker-compose.temporal.yml down
```

## Run App in Development Mode

To run the application directly (without Makefile or Docker), ensure all required environment variables are set. You can use a `.env` file or export them manually.

Start the app with:

```sh
go run cmd/main.go <option>
```

Replace `<option>` with any command-line arguments your application supports. For example:

```sh
go run cmd/main.go http
```

Make sure external dependencies (such as PostgreSQL, RabbitMQ, and Redis) are running, either via Docker Compose or another method.

## CLI Commands

The project uses [`prabogo-cli`](https://github.com/prabogo/prabogo-cli) for code generation and development tasks. Install it with:

```sh
go install github.com/prabogo/prabogo-cli@latest
```

### Interactive Command Runner

![Alt text](./docs/images/option.gif "Option")

You can use the interactive target selector to choose and run targets:

```sh
prabogo-cli run
```

This will display an interactive menu to select a target and will prompt for any required parameters. The selector works in two modes:

1. If `fzf` is installed: Uses a fuzzy-search interactive selector (recommended for best experience)
2. If `fzf` is not available: Falls back to a basic numbered menu selection

To install `fzf` (optional):
- macOS: `brew install fzf`
- Linux: `apt install fzf` (Ubuntu/Debian) or `dnf install fzf` (Fedora)
- Windows: With chocolatey: `choco install fzf` or with WSL, follow Linux instructions

### Common CLI Targets

#### Code Generation Targets

- `model`: Creates a model/entity with necessary structures
  ```sh
  prabogo-cli model name
  ```

- `migration-postgres`: Creates a PostgreSQL migration file
  ```sh
  prabogo-cli migration-postgres name
  ```

- `inbound-http-fiber`: Creates HTTP handlers using Fiber framework
  ```sh
  prabogo-cli inbound-http-fiber name
  ```

- `inbound-message-rabbitmq`: Creates RabbitMQ message consumers
  ```sh
  prabogo-cli inbound-message-rabbitmq name
  ```

- `inbound-command`: Creates command line interface handlers
  ```sh
  prabogo-cli inbound-command name
  ```

- `inbound-workflow-temporal`: Creates Temporal workflow worker
  ```sh
  prabogo-cli inbound-workflow-temporal name
  ```

- `outbound-database-postgres`: Creates PostgreSQL database adapter
  ```sh
  prabogo-cli outbound-database-postgres name
  ```

- `outbound-http`: Creates HTTP adapter
  ```sh
  prabogo-cli outbound-http name
  ```

- `outbound-message-rabbitmq`: Creates RabbitMQ message publisher adapter
  ```sh
  prabogo-cli outbound-message-rabbitmq name
  ```

- `outbound-cache-redis`: Creates Redis cache adapter
  ```sh
  prabogo-cli outbound-cache-redis name
  ```

- `outbound-workflow-temporal`: Creates Temporal workflow starter adapter
  ```sh
  prabogo-cli outbound-workflow-temporal name
  ```

- `generate-mocks`: Generates mock implementations from all go:generate directives in registry files
  ```sh
  prabogo-cli generate-mocks
  ```

#### Runtime Targets

- `build`: Builds the Docker image for the application
  ```sh
  prabogo-cli build
  # Force rebuild regardless of existing image:
  prabogo-cli build --build
  ```

- `http`: Runs the application in HTTP server mode inside Docker
  ```sh
  prabogo-cli http
  # Force rebuild before running:
  prabogo-cli http --build
  ```

- `message`: Runs the application in message consumer mode inside Docker
  ```sh
  prabogo-cli message upsert_client
  # Force rebuild before running:
  prabogo-cli message upsert_client --build
  ```

- `command`: Executes a specific command in the application
  ```sh
  prabogo-cli command publish_upsert_client name
  # Force rebuild before running:
  prabogo-cli command publish_upsert_client name --build
  ```

- `workflow`: Runs the application in workflow worker mode inside Docker
  ```sh
  prabogo-cli workflow upsert_client
  # Force rebuild before running:
  prabogo-cli workflow upsert_client --build
  ```

## Running test suite

### Unit tests

```sh
go test -cover ./internal/domain/...
```

To generate coverage report:

```sh
go test -coverprofile=coverage.profile -cover ./internal/domain/...
go tool cover -html coverage.profile -o coverage.html
```

Coverage report will be available at `coverage.html`

To check intermittent test failure due to mock. when in doubt, use `-t 1000`
```sh
retry -d 0 -t 100 -u fail -- go test -coverprofile=coverage.profile -cover ./internal/domain/... -count=1
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Author

Moch Dieqy Dzulqaidar