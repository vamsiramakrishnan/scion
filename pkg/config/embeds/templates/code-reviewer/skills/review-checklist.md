# Code Review Checklist

When reviewing code, check each of these systematically:

## Correctness
- [ ] Logic handles edge cases (empty inputs, zero values, nil/null)
- [ ] Error paths return appropriate errors (not swallowed)
- [ ] Concurrency safety (shared state protected, no data races)
- [ ] Resource cleanup (files closed, connections returned to pool)

## Security
- [ ] User input validated before use
- [ ] SQL queries parameterized (no string concatenation)
- [ ] Auth checks on all sensitive operations
- [ ] No secrets in code or logs

## Performance
- [ ] No N+1 query patterns
- [ ] Appropriate use of indices for database queries
- [ ] No unbounded memory growth (pagination, streaming)
- [ ] Hot paths avoid unnecessary allocations

## Maintainability
- [ ] Functions have single responsibility
- [ ] Names communicate intent
- [ ] Tests cover the critical path
- [ ] No dead code or commented-out blocks
