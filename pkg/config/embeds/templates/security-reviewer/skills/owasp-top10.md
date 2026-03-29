# OWASP Top 10 Security Checklist

When performing a security audit, check for each of these vulnerability categories:

## A01: Broken Access Control
- Verify authorization on every endpoint (not just authentication)
- Check for IDOR (Insecure Direct Object References)
- Ensure principle of least privilege for roles and permissions
- Check CORS configuration is restrictive

## A02: Cryptographic Failures
- Verify sensitive data is encrypted at rest and in transit
- Check for weak algorithms (MD5, SHA1 for passwords)
- Ensure TLS is enforced (no HTTP fallback)
- Check for hardcoded keys or secrets

## A03: Injection
- SQL: Verify parameterized queries everywhere
- Command: Check for shell exec with user input
- XSS: Verify output encoding/escaping
- LDAP, NoSQL, XML injection variants

## A04: Insecure Design
- Check for missing rate limiting
- Verify business logic constraints (negative quantities, etc.)
- Check for missing account lockout

## A05: Security Misconfiguration
- Default credentials or settings
- Unnecessary features enabled
- Missing security headers
- Verbose error messages exposing internals

## A06: Vulnerable Components
- Check dependency versions against known CVEs
- Look for abandoned/unmaintained dependencies

## A07: Authentication Failures
- Weak password requirements
- Missing MFA
- Session fixation or improper invalidation
- JWT algorithm confusion

## A08: Data Integrity Failures
- Verify input validation on deserialization
- Check for unsigned/unverified updates
- CI/CD pipeline security

## A09: Logging & Monitoring Failures
- Verify audit logging for sensitive operations
- Check that logs don't contain secrets/PII
- Ensure alerting for security events

## A10: Server-Side Request Forgery (SSRF)
- Validate and sanitize all URLs from user input
- Use allowlists for external service access
- Block access to internal network ranges
