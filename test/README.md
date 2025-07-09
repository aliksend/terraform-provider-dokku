# Test Configurations

This directory contains different test configurations for the Dokku Terraform provider.

## Test Structure

- **test_helpers.go**: Shared infrastructure for all E2E tests (Docker setup, SSH setup, validation functions)
- **individual test files**: Each test has its own `*_test.go` file with native Go test functions and inline configurations
- **Test subdirectories**: Contains specific Terraform configurations for different test scenarios

## Test Execution

**Important**: Tests run **sequentially** (not in parallel) because they share Docker container resources. Each test:
1. Starts its own Dokku Docker container
2. Runs the test scenario
3. Cleans up the container before the next test begins

## Test Types

### Core Provider Tests

#### Simple Configuration (test/simple/)
- **Test Function**: `TestSimple`
- **Purpose**: Basic dokku_app resource with minimal configuration
- **Features**: Simple Docker image deployment with nginx:alpine
- **Configuration**: Inline configuration with `appName := "simple-test-app"`
- **Use case**: Testing basic provider functionality
- **Status**: ✅ **PASSES** - Provider SSH connection issue fixed

#### Complex App Configuration (test/complex_app_config/)
- **Test Function**: `TestComplexAppConfig`
- **Purpose**: Advanced dokku_app resource with complex configuration
- **Features**: 
  - Complex environment variables (ENV=prod, NODE_ENV=production, etc.)
  - Multiple domains (primary and API domains)
  - Hardcoded configuration in main.tf to avoid Terraform map issues
- **Configuration**: Inline configuration with `appName := "complex-test-app"`
- **Use case**: Testing complex application deployments with multiple settings
- **Status**: ❌ **FAILS** - Provider SSH connection issue: "unable go get dokku version"

### Example-Based Tests

These tests use actual content from `examples/resources/` to ensure examples work correctly:

#### Dokku App Example
- **Test Function**: `TestExampleDokkuApp`
- **Purpose**: Tests real dokku_app example configuration
- **Features**: Uses actual examples/resources/dokku_app content with complex configurations
- **Status**: ❌ **FAILS** - Provider SSH connection issue: "unable go get dokku version"

#### Dokku Domain Example  
- **Test Function**: `TestExampleDokkuDomain`
- **Purpose**: Tests real dokku_domain example configuration
- **Features**: Uses actual examples/resources/dokku_domain content
- **Status**: ❌ **FAILS** - Provider SSH connection issue: "unable go get dokku version"

#### Plugin-Dependent Examples

These tests automatically install required plugins during test execution:

- **Test Function**: `TestExampleDokkuHttpAuth`
- **Purpose**: Tests dokku_http_auth example
- **Features**: Automatically installs http-auth plugin via `dokku plugin:install`
- **Status**: ❌ **FAILS** - Provider SSH connection issue: "unable go get dokku version"

- **Test Function**: `TestExampleDokkuPlugin`
- **Purpose**: Tests dokku_plugin example
- **Features**: Tests plugin management functionality
- **Status**: ❌ **FAILS** - Provider SSH connection issue: "unable go get dokku version"

- **Test Function**: `TestExampleDokkuPostgres`
- **Purpose**: Tests dokku_postgres example
- **Features**: Automatically installs postgres plugin via `dokku plugin:install`
- **Status**: ❌ **FAILS** - Provider SSH connection issue: "unable go get dokku version"

## Running Tests

### Individual Tests (Recommended)

Tests run sequentially and clean up between executions:

```bash
cd test

# Core provider tests
go test -v -run "^TestSimple$"                    # ✅ PASSES
go test -v -run "^TestComplexAppConfig$"         # ❌ FAILS (provider connection issue)

# Example-based tests (currently failing due to provider connection issue)
go test -v -run "^TestExampleDokkuApp$"          # ❌ FAILS (provider connection issue)
go test -v -run "^TestExampleDokkuDomain$"       # ❌ FAILS (provider connection issue)

# Plugin-dependent tests (currently failing due to provider connection issue)
go test -v -run "^TestExampleDokkuHttpAuth$"     # ❌ FAILS (provider connection issue)
go test -v -run "^TestExampleDokkuPlugin$"       # ❌ FAILS (provider connection issue)
go test -v -run "^TestExampleDokkuPostgres$"     # ❌ FAILS (provider connection issue)
```

### All Tests Sequentially

```bash
cd test
go test -v ./...  # Runs all tests sequentially (Docker containers will not conflict)
```

### Running Specific Test Patterns

```bash
cd test
go test -v -run "TestExample"  # Runs all example tests
go test -v -run "TestSimple|TestComplexAppConfig"  # Runs core provider tests
```

### Direct Terraform Usage (Legacy)

You can still run Terraform directly for manual testing:

#### Simple Configuration
```bash
cd test/simple
terraform init
terraform plan -var="ssh_private_key=$(cat ~/.ssh/id_rsa)"
terraform apply -var="ssh_private_key=$(cat ~/.ssh/id_rsa)"
```

#### Complex App Configuration
```bash
cd test/complex_app_config
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your values
terraform init
terraform plan
terraform apply
```

## Known Issues

### Provider SSH Connection Issue (RESOLVED)

~~All tests currently fail with the same error: `"unable go get dokku version"`.~~ **FIXED**: This issue was resolved by correcting the provider's `GetVersion` function in `provider/dokku_client/client.go:112` from calling `c.RunQuiet(ctx, "--version")` to `c.RunQuiet(ctx, "version")`.

### Provider Connection Issue (PARTIAL FIX)

Most tests fail with `"unable go get dokku version"` error. While this was fixed for simple configurations (TestSimple passes), complex configurations and example tests still trigger this issue, indicating there are multiple code paths that need the same fix or additional connection issues to resolve.

## Test Development

### Adding New Tests

1. Create a new `*_test.go` file
2. Implement a `TestXxx` function with inline configuration using the terratest framework
3. Follow the pattern of existing tests for Docker setup, SSH configuration, and Terraform execution
4. Add the test to the GitHub workflow matrix
5. Update this README

### Shared Infrastructure

All shared code is in `test_helpers.go`:
- Docker container setup and cleanup
- SSH key generation and configuration  
- Terraform provider building
- Validation functions
- Helper utilities

Individual test files contain complete self-contained test functions with inline configurations.

### Test Architecture

Each test follows this pattern:
1. **Generate SSH Keys**: Creates unique SSH keys for the test
2. **Setup Docker**: Starts a fresh Dokku container
3. **Setup SSH**: Configures SSH access to the container
4. **Apply Terraform**: Runs terraform apply with inline configuration
5. **Validate**: Checks that resources were created correctly via SSH
6. **Cleanup**: Destroys Terraform resources and removes Docker container

## CI/CD Integration

Tests run in GitHub Actions with a matrix strategy:
- Each test runs in a separate VM (no Docker container conflicts)
- Tests can run in parallel across different runners  
- Uses `-run TestXxx` for native Go test filtering
- Each test is completely self-contained and independent