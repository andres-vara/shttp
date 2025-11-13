Title: router: add path-params extraction; middleware improvements

Description:
This PR implements minimal path-parameter extraction without adding third-party dependencies, and improves middleware logging and API compatibility.

Changes included:
- Add shttp/pathparams.go with helpers: SetPathValues, SetPathValue, PathValue, extractPathParams.
- Update shttp/router.go to inject path params into the request context for route patterns using {param}.
- Add LoggerMiddleware(logger) and update LoggingMiddleware(logger) to accept an optional logger and emit structured [http.request] / [http.response] logs including request_id, user_id, client_ip, status, duration_ms, and error when present.
- Update tests and examples to use the new helpers.
- Add GitHub Actions workflow .github/workflows/ci.yml to run tests for both slogr and shttp modules on push and PRs.

Notes:
- Path extraction uses simple segment-by-segment matching; named tokens must be full path segments (for example /users/{id}). No wildcard/regex support yet to keep the implementation small and dependency-free.
- Tests: ran go test ./... -v locally; all tests passed.

Follow-ups:
- Extend path pattern support if desired (wildcards/optional segments).
- Add a short README usage example showing PathValue usage.
- Add an integration test that starts the server on port 0 to exercise middleware end-to-end.
