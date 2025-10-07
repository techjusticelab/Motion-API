# Test Directory (`/test`)

This directory contains testing utilities, integration tests, and test files that don't belong in specific package directories. It supports the project's comprehensive testing strategy following Test-Driven Development (TDD) principles.

## Structure

```
test/
└── unit/               # Unit test files and utilities
    └── search/         # Search-specific unit tests
        └── client_test.go
```

## Testing Philosophy

### Test-Driven Development (TDD)
The Motion-Index Fiber project follows strict TDD principles:
1. **Red**: Write failing tests first
2. **Green**: Write minimal code to make tests pass
3. **Refactor**: Improve code while keeping tests green

### Testing Hierarchy
1. **Unit Tests**: Fast, isolated tests with no external dependencies
2. **Integration Tests**: Test interactions between components
3. **End-to-End Tests**: Full system testing with real services
4. **Benchmark Tests**: Performance validation and optimization

## Test Categories

### Unit Tests (`/unit`)
**Purpose**: Fast, isolated testing of individual components
**Location**: Alongside source code (`*_test.go`) and in `/test/unit/`
**Characteristics**:
- No external dependencies
- Use mocks for external services
- Fast execution (< 1 second)
- High code coverage

**Example Structure**:
```
test/unit/
├── search/           # Search component unit tests
├── storage/          # Storage component unit tests
├── processing/       # Processing component unit tests
└── handlers/         # Handler unit tests
```

### Integration Tests
**Purpose**: Test interactions between real components
**Location**: `*_integration_test.go` files throughout codebase
**Characteristics**:
- Use real external services when possible
- Test actual API interactions
- Validate end-to-end workflows
- Require proper environment setup

**Build Tags**: Use `// +build integration` to separate from unit tests

### Benchmark Tests
**Purpose**: Performance validation and optimization
**Location**: `*_benchmark_test.go` or `*_test.go` with benchmark functions
**Characteristics**:
- Measure execution time and memory allocation
- Track performance regressions
- Optimize critical code paths
- Compare implementation strategies

## Test Utilities and Helpers

### Common Test Patterns
1. **Setup/Teardown**: Consistent test environment management
2. **Mock Factories**: Reusable mock object creation
3. **Test Data**: Consistent test data generation
4. **Assertions**: Custom assertion helpers
5. **Environment**: Test environment configuration

### Test Configuration
```go
// Example test configuration
type TestConfig struct {
    TempDir     string
    MockStorage bool
    TestDB      string
    LogLevel    string
}
```

## Running Tests

### All Tests
```bash
# Run all tests with coverage
go test ./... -v -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Unit Tests Only
```bash
# Fast feedback loop
go test ./... -short -v
```

### Integration Tests
```bash
# Requires real service credentials
RUN_INTEGRATION_TESTS=true go test ./... -v -tags=integration
```

### Specific Test Categories
```bash
# Test specific packages
go test ./test/unit/search/... -v
go test ./internal/handlers/... -v
go test ./pkg/processing/... -v
```

### Benchmark Tests
```bash
# Run performance benchmarks
go test ./... -bench=. -benchmem
```

### Race Condition Detection
```bash
# Detect race conditions
go test ./... -race -v
```

## Test Data Management

### Test Fixtures
- Sample documents for processing tests
- Mock API responses
- Test configuration files
- Reference data for validation

### Temporary Resources
- Temporary file management
- Test database cleanup
- Mock service lifecycle
- Resource isolation

## Testing Best Practices

### Unit Test Guidelines
1. **Isolation**: Each test should be independent
2. **Fast**: Unit tests should complete quickly
3. **Deterministic**: Tests should produce consistent results
4. **Focused**: Test one behavior per test function
5. **Clear**: Test names should describe the behavior being tested

### Integration Test Guidelines
1. **Real Services**: Use actual external services when possible
2. **Environment**: Clearly document required test environment
3. **Cleanup**: Ensure proper cleanup of test resources
4. **Error Scenarios**: Test both success and failure cases
5. **Performance**: Include performance validation

### Mock Usage
1. **Interfaces**: Mock external dependencies via interfaces
2. **Behavior**: Mock realistic service behavior
3. **Edge Cases**: Mock error conditions and edge cases
4. **Simplicity**: Keep mocks simple and focused
5. **Validation**: Verify mock interactions when necessary

## Test Coverage Goals

### Coverage Targets
- **Unit Tests**: > 80% line coverage
- **Integration Tests**: Critical path coverage
- **End-to-End Tests**: Key user scenarios
- **Benchmark Tests**: Performance-critical functions

### Coverage Analysis
```bash
# Generate detailed coverage report
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## Continuous Integration

### Automated Testing
- All tests run on every pull request
- Integration tests with real services
- Performance regression detection
- Code coverage reporting

### Test Environment
- Isolated test environments for each build
- Proper secret management for integration tests
- Cleanup of test resources
- Parallel test execution where safe

## Debugging Tests

### Common Issues
1. **Flaky Tests**: Investigate timing and race conditions
2. **Resource Leaks**: Check for proper cleanup
3. **Mock Behavior**: Verify mock setup and expectations
4. **Environment**: Validate test environment configuration

### Debugging Tools
```bash
# Verbose test output
go test -v ./...

# Debug specific test
go test -v -run TestSpecificFunction

# Test with debugging info
go test -v -args -debug
```

## Contributing to Tests

### Adding New Tests
1. Follow TDD principles (write test first)
2. Use appropriate test category (unit/integration/benchmark)
3. Include error case testing
4. Maintain consistent naming conventions
5. Update documentation for complex test scenarios

### Test Maintenance
1. Keep tests up-to-date with code changes
2. Remove obsolete tests
3. Refactor test code for maintainability
4. Monitor test execution performance
5. Review test coverage regularly