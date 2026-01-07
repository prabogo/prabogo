# Security Guidelines for Prabogo

This document outlines security best practices for deploying Prabogo in production environments.

## Production Deployment Checklist

### 1. Database SSL/TLS

By default, `DATABASE_SSLMODE=disable` is set for local development. In production:

```env
# Options (from least to most secure):
# - require: Encrypt connection but don't verify server certificate
# - verify-ca: Encrypt and verify server certificate is signed by trusted CA
# - verify-full: Encrypt, verify CA, and verify hostname matches certificate
DATABASE_SSLMODE=verify-full
```

### 2. Internal API Key

Generate a cryptographically secure key for `INTERNAL_KEY`:

```bash
# Generate a 32-byte (64 hex characters) random key
openssl rand -hex 32
```

- Minimum recommended length: 32 characters
- Rotate keys periodically
- Never commit keys to version control

### 3. Secrets Management

For production deployments, use a secrets manager instead of environment files:

- **AWS**: AWS Secrets Manager or Parameter Store
- **GCP**: Secret Manager
- **Azure**: Key Vault
- **Self-hosted**: HashiCorp Vault

### 4. Docker Security

- Use specific image tags (avoid `latest`)
- Run containers as non-root user
- Use Docker secrets for sensitive data in Swarm/Compose
- Scan images for vulnerabilities

### 5. Network Security

- Use TLS for all external connections (database, Redis, RabbitMQ)
- Deploy behind a reverse proxy (nginx, traefik) with HTTPS
- Implement rate limiting at the proxy level
- Use private networks for service-to-service communication

### 6. Application Security

- Keep dependencies updated (`go get -u`)
- Run security scanners (gosec, govulncheck)
- Enable structured logging in production (`APP_MODE=release`)
- Implement request validation and sanitization

## Environment Files

| File | Purpose | Git Status |
|------|---------|------------|
| `.env` | Runtime configuration | **Never commit** |
| `.env.example` | Template with placeholders | Safe to commit |
| `.env.docker` | Docker Compose overrides | **Never commit** |
| `.env.docker.example` | Docker template | Safe to commit |

## Reporting Security Issues

If you discover a security vulnerability, please report it responsibly by contacting the maintainers directly rather than opening a public issue.
