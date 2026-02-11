# Contributing to Reloquent

Thank you for your interest in contributing to Reloquent. This guide covers everything you need to get started.

## Reporting Bugs and Requesting Features

- **Bugs:** Open a GitHub Issue with the "bug" label. Include the Reloquent version (`reloquent --version`), your OS, the source database type and version, and steps to reproduce the problem. If possible, include the relevant section of your `reloquent.yaml` (with secrets redacted).
- **Feature requests:** Open a GitHub Issue with the "enhancement" label. Describe the use case, the expected behavior, and why it would benefit other users.
- **Questions:** Use GitHub Discussions for general questions about usage or architecture.

## Development Setup

### Prerequisites

- **Go 1.25+** for the backend and CLI
- **Node.js 20+** and npm for the web UI
- **Docker and Docker Compose** for integration tests and the trial environment
- **Make** for build and test commands

### Clone and Build

```bash
git clone https://github.com/reloquent/reloquent.git
cd reloquent
make build
```

This compiles the Go binary (including the embedded web UI assets) and places it in `bin/`.

### Run Tests

```bash
# Unit tests
make test

# Integration tests (starts Docker containers for PostgreSQL, Oracle, and MongoDB)
make test-integration
```

### Run Locally

```bash
# Run the CLI directly
go run ./cmd/reloquent

# Start the web UI in development mode (hot reload)
cd web && npm install && npm run dev
```

The development web UI proxies API requests to the Go backend. Start the backend separately with `go run ./cmd/reloquent serve` when developing the frontend.

## Code Style

### Go

- Format all code with `gofmt`.
- Run `go vet` and `golint` before committing. The CI pipeline enforces these checks.
- Follow standard Go conventions. Use `internal/` packages for non-exported code.
- Write table-driven tests. Place test files alongside the code they test (`foo_test.go` next to `foo.go`).

### React / TypeScript

- Use functional components with hooks.
- Style with Tailwind CSS utility classes.
- Write TypeScript (not JavaScript) for all new code.
- Test with React Testing Library.

### General

- Write tests alongside each feature, not after. Every PR that adds or changes behavior should include corresponding tests.
- Keep functions focused and files reasonably sized.
- Add comments for non-obvious logic, but prefer clear naming over excessive commentary.

## Testing

Reloquent uses multiple layers of testing:

| Layer | Command | Description |
|---|---|---|
| Unit tests | `make test` | Fast, no external dependencies. Test core engine logic, YAML parsing, PySpark codegen, and schema operations. |
| Integration tests | `make test-integration` | Use Docker Compose to spin up real PostgreSQL, Oracle (XE), and MongoDB instances. Test end-to-end discovery, migration, and validation. |
| Web UI tests | `cd web && npm test` | React Testing Library tests for the frontend components. |

When adding a new feature, determine which layers are appropriate and add tests at each one.

## Pull Request Process

1. **Fork the repository** and create a feature branch from `main`.

   ```bash
   git checkout -b feat/your-feature-name
   ```

2. **Make your changes.** Follow the code style guidelines above and write tests for new functionality.

3. **Use Conventional Commits** for all commit messages (see format below).

4. **Ensure all tests pass** before pushing.

   ```bash
   make test
   cd web && npm test
   ```

5. **Push your branch** and open a pull request against `main`.

   ```bash
   git push origin feat/your-feature-name
   ```

6. **Fill out the PR template.** Describe what the change does, why it is needed, and how it was tested.

7. **Respond to review feedback.** Maintainers may request changes. Push additional commits to the same branch rather than force-pushing.

8. Once approved and CI passes, a maintainer will merge the PR.

## Commit Message Format

Reloquent uses [Conventional Commits](https://www.conventionalcommits.org/). Every commit message must follow this format:

```
<type>: <short description>

[optional body]

[optional footer(s)]
```

### Types

| Type | When to use |
|---|---|
| `feat` | A new feature or capability |
| `fix` | A bug fix |
| `test` | Adding or updating tests (no production code change) |
| `docs` | Documentation only changes |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `chore` | Build process, dependency updates, tooling |
| `ci` | CI/CD configuration changes |

### Examples

```
feat: add Oracle schema discovery with partition detection

fix: handle empty tables during row count validation

test: add integration tests for PySpark codegen with nested arrays

docs: add configuration reference to README
```

## License

By contributing to Reloquent, you agree that your contributions will be licensed under the [BSD 3-Clause License](LICENSE).
