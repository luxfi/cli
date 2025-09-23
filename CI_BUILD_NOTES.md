# CI Build Notes for luxfi/cli

## Current Status

The CLI repository builds successfully locally but has dependencies on private luxfi repositories that are not published to public module registries.

## Build Requirements

### Local Development
- Uses local replace directives in go.mod for luxfi packages
- Build command: `make build`
- Test command: `make test`

### CI/CD Configuration
- **IMPORTANT**: CI builds currently fail due to private repository dependencies
- Required environment variables:
  - `GOSUMDB=off` - Disable checksum verification
  - `GOPROXY=direct` - Use direct module fetching

## Dependencies Issue

The following luxfi packages are required but not available in public registries:
- `github.com/luxfi/node v1.17.1`
- `github.com/luxfi/sdk v1.0.0`
- `github.com/luxfi/netrunner v1.13.5-lux.2`
- `github.com/luxfi/evm v1.16.18`

## Solution Options

1. **Publish packages to GitHub** (Recommended)
   - Tag all dependent repositories with proper semantic versions
   - Ensure they are publicly accessible or configure GitHub Actions with appropriate tokens

2. **Use vendoring**
   - Run `go mod vendor` to include dependencies
   - Commit vendor directory (large but ensures reproducible builds)

3. **Private module proxy**
   - Set up Athens or similar proxy for private modules
   - Configure CI to use the proxy

## Current Workaround

For local development, the go.mod file includes replace directives pointing to local directories. These MUST be removed before pushing to CI.

## Test Status

Several test packages have compilation errors due to API changes in dependencies. These need to be fixed:
- `pkg/apmintegration` - undefined luxlog.NoWarn
- `cmd/flags` - interface mismatch with mocks.Prompter
- `pkg/prompts/capturetests` - interface method signature changes
- `pkg/plugins` - undefined config variables

## Semantic Versioning

Latest tag: `v1.9.2-lux.3`

Follows semantic versioning with `-lux.X` suffix for Lux-specific releases.