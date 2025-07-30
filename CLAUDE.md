# CLAUDE.md

## Build/Lint/Test Commands
- Build & Deploy plugin: `make deploy`
- Lint code and fix some errors, will edit files if fixes needed: `make check-style-fix`
- Run all tests: `make test`
- Run specific Go test: `go test -v ./server/path/to/package -run TestName`
- Run e2e tests: `make e2e`
- Run specific e2e test file: `cd e2e && npx playwright test filename.spec.ts --reporter=list`

## Code Style Guidelines
- Go: Follow Go standard formatting conventions according to goimports
- TypeScript/React: Use 4-space indentation, PascalCase for components, strict typing, always use styled-components, never use style properties
- Error handling: Check all errors explicitly in production code
- File naming: Use snake_case for file names
- Documentation: Include license header in all files
- Use descriptive variable and function names
- Use small, focused functions
- Write go unit tests whenever possible
- Never use mocking or introduce new testing libraries
- Document all public APIs
- Always add i18n for new text
- Write go unit tests as table driven tests whenever possible
