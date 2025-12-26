# Contributing to nuke

Thank you for considering contributing to nuke! This document provides guidelines for contributing to the project.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/nuke_rm.git`
3. Create a new branch: `git checkout -b feature/your-feature-name`
4. Make your changes
5. Run tests: `make test`
6. Commit your changes: `git commit -m "Add your feature"`
7. Push to your fork: `git push origin feature/your-feature-name`
8. Create a Pull Request

## Development Setup

### Prerequisites

- Go 1.21 or later
- Make (optional but recommended)
- golangci-lint (for linting)

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Run linter
make lint
```

## Code Style

- Follow standard Go conventions
- Run `gofmt` before committing
- Use meaningful variable and function names
- Add comments for exported functions and types

## Testing

- Write tests for new features
- Ensure existing tests pass before submitting PR
- Aim for good test coverage

## Pull Request Guidelines

1. **Title**: Use a clear, descriptive title
2. **Description**: Explain what changes you made and why
3. **Tests**: Include tests for new features
4. **Documentation**: Update README if needed
5. **Single Purpose**: Keep PRs focused on one thing

## Reporting Issues

When reporting issues, please include:

- OS and version
- Go version
- Steps to reproduce
- Expected vs actual behavior
- Any error messages

## Feature Requests

Feature requests are welcome! Please:

1. Check if the feature already exists or is planned
2. Provide a clear use case
3. Be open to discussion about implementation

## Code of Conduct

- Be respectful and inclusive
- Welcome newcomers
- Focus on constructive feedback
- No harassment or discrimination

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
