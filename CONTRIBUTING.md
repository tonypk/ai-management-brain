# Contributing to AI Management Brain

Thank you for your interest in contributing! We welcome bug reports, feature requests, and pull requests.

## Development Setup

### Prerequisites
- Go 1.22+
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+

### Quick Start

1. Clone the repository:
```bash
git clone https://github.com/tonypk/ai-management-brain.git
cd ai-management-brain
```

2. Copy the environment file:
```bash
cp .env.example .env
```

3. Generate required keys:
```bash
openssl rand -hex 32  # for ENCRYPTION_KEY
openssl rand -hex 32  # for JWT_SECRET
```

4. Start services:
```bash
docker compose up -d
```

5. Run migrations:
```bash
go run ./cmd/brain migrate
```

## Development Workflow

### Running Tests
```bash
go test ./... -v -race
```

### Building
```bash
go build ./...
```

### Linting
```bash
go vet ./...
```

### Code Style

We follow Go conventions:
- Use `gofmt` for formatting
- Use descriptive variable names
- Keep functions focused and under 50 lines
- Add comments for exported functions
- Use sqlc for database access (see `sql/` directory)

## Database Changes

For schema changes:
1. Add migration in `sql/migrations/`
2. Update queries in `sql/queries/`
3. Run `sqlc generate` to update Go code
4. Test migrations work forward and backward

## Pull Request Process

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/your-feature`
3. Make your changes
4. Write tests for new functionality
5. Run `go test ./...` to ensure tests pass
6. Commit with clear messages (see [Conventional Commits](https://www.conventionalcommits.org/))
7. Push to your fork and create a Pull Request

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):
- `feat:` new feature
- `fix:` bug fix
- `refactor:` code refactoring
- `test:` tests
- `docs:` documentation
- `chore:` maintenance

Example:
```
feat: add task prioritization endpoint

Implements new /api/v1/tasks/{id}/priority endpoint with support
for reordering tasks within a board.
```

## Questions?

Feel free to open a GitHub issue or discussion for questions or suggestions.

Happy coding!
