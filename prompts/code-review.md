Review this Go API project as a senior Go developer. Walk through every .go file and analyze the codebase across these dimensions:

1. **Code Organization & Idiomatic Go**
   - Package structure and naming conventions
   - Interface usage and dependency injection
   - Error handling patterns (wrapping, sentinel errors, custom types)
   - Proper use of context.Context propagation
   - Goroutine lifecycle management and leak prevention

2. **Security**
   - SQL injection or other injection vectors
   - Input validation and sanitization
   - Authentication/authorization implementation
   - Secrets management (hardcoded credentials, env handling)
   - CORS, rate limiting, and header security
   - TLS configuration if applicable
   - Timeout settings on HTTP server and clients

3. **Concurrency & Performance**
   - Race conditions or unsafe shared state
   - Proper use of sync primitives (Mutex, WaitGroup, channels)
   - Connection pool configuration (DB, HTTP clients)
   - Memory allocations that could be reduced (unnecessary copies, missing pointer receivers)
   - N+1 query patterns or missing database indexes hints

4. **Reliability & Observability**
   - Graceful shutdown handling
   - Structured logging practices
   - Health check endpoints
   - Panic recovery middleware
   - Resource cleanup (defer Close, connection leaks)

5. **Testing & Maintainability**
   - Test coverage gaps and testability of the design
   - Missing table-driven tests for complex logic
   - Mocking strategy (interfaces vs concrete types)
   - API versioning approach

6. **Dependency & Configuration**
   - Go module hygiene (unused deps, pinned versions)
   - Configuration management (env vars, config files, defaults)
   - Third-party library choices (well-maintained, necessary)

For each issue found, provide:
- The file and line reference
- Severity: CRITICAL / HIGH / MEDIUM / LOW
- A brief explanation of why it matters
- A concrete code fix or recommendation
