# CI/CD Pipeline Documentation

## Overview

The BIG SKIES Framework uses GitHub Actions for comprehensive CI/CD automation, ensuring code quality, security, and reliable deployments.

## Workflows

### 1. CI Pipeline (`ci.yml`)
**Triggers**: Push/PR to main/develop branches
**Purpose**: Comprehensive testing and validation

- **Unit Tests**: Go test suite with race detection
- **Integration Tests**: Full stack testing with PostgreSQL and MQTT
- **Linting**: golangci-lint for code quality
- **Docker Build**: Multi-stage optimized builds
- **Security Scanning**: Trivy vulnerability scanning

### 2. Release Pipeline (`release.yml`)
**Triggers**: Git tags (v*), manual dispatch
**Purpose**: Automated Docker image building and publishing

- **Multi-platform Builds**: Linux AMD64/ARM64
- **Docker Hub Publishing**: Automated image publishing
- **GitHub Releases**: Tagged release creation

### 3. Security Scanning (`security.yml`)
**Triggers**: Push/PR, weekly schedule
**Purpose**: Security vulnerability detection

- **CodeQL Analysis**: Static application security testing (SAST)
- **Dependency Review**: Open source dependency vulnerabilities
- **Trivy Container Scanning**: Docker image vulnerability scanning

### 4. Code Quality (`quality.yml`)
**Triggers**: Push/PR to main/develop
**Purpose**: Code quality and coverage analysis

- **golangci-lint**: Comprehensive Go linting
- **Test Coverage**: 80% minimum threshold
- **SonarCloud**: Code quality metrics and analysis
- **Security Checks**: govulncheck for known vulnerabilities

### 5. Performance Testing (`performance.yml`)
**Triggers**: Push/PR, weekly schedule
**Purpose**: Performance regression detection

- **Benchmark Tests**: Go benchmark suite
- **Load Testing**: HTTP endpoint load testing with hey
- **Performance Tracking**: GitHub Actions benchmark storage

### 6. Documentation (`docs.yml`)
**Triggers**: Changes to docs/, README.md, *.md
**Purpose**: Documentation validation and deployment

- **Link Checking**: Validate all documentation links
- **API Documentation**: Auto-generate Swagger docs
- **GitHub Pages**: Deploy documentation site

### 7. Deployment (`deploy.yml`)
**Triggers**: Push to main, tags, manual dispatch
**Purpose**: Automated staging/production deployment

- **Staging Deployment**: Automated staging environment updates
- **Production Deployment**: Zero-downtime production releases
- **Smoke Tests**: Post-deployment health validation

### 8. Health Monitoring (`health-check.yml`)
**Triggers**: 6-hour schedule, manual dispatch
**Purpose**: CI/CD pipeline health monitoring

- **Workflow Status**: Monitor recent workflow success/failure rates
- **Branch Protection**: Validate branch protection rules
- **Dependency Updates**: Check for outdated dependencies
- **Alerting**: Automated notifications for pipeline issues

## Configuration Files

### Dependabot (`dependabot.yml`)
Automated dependency updates for:
- Go modules (weekly, Mondays)
- Docker images (weekly, Mondays)
- GitHub Actions (weekly, Mondays)

### Link Check Config (`link-check-config.json`)
Configuration for documentation link validation:
- Ignore localhost URLs
- Custom timeout and retry settings
- URL replacement patterns

## Status Badges

Add these badges to your README.md:

```markdown
[![CI](https://github.com/unklstewy/BIG_SKIES_FRAMEWORK/actions/workflows/ci.yml/badge.svg)](https://github.com/unklstewy/BIG_SKIES_FRAMEWORK/actions/workflows/ci.yml)
[![Release](https://github.com/unklstewy/BIG_SKIES_FRAMEWORK/actions/workflows/release.yml/badge.svg)](https://github.com/unklstewy/BIG_SKIES_FRAMEWORK/actions/workflows/release.yml)
[![Security](https://github.com/unklstewy/BIG_SKIES_FRAMEWORK/actions/workflows/security.yml/badge.svg)](https://github.com/unklstewy/BIG_SKIES_FRAMEWORK/actions/workflows/security.yml)
[![Quality](https://github.com/unklstewy/BIG_SKIES_FRAMEWORK/actions/workflows/quality.yml/badge.svg)](https://github.com/unklstewy/BIG_SKIES_FRAMEWORK/actions/workflows/quality.yml)
[![Performance](https://github.com/unklstewy/BIG_SKIES_FRAMEWORK/actions/workflows/performance.yml/badge.svg)](https://github.com/unklstewy/BIG_SKIES_FRAMEWORK/actions/workflows/performance.yml)
[![Documentation](https://github.com/unklstewy/BIG_SKIES_FRAMEWORK/actions/workflows/docs.yml/badge.svg)](https://github.com/unklstewy/BIG_SKIES_FRAMEWORK/actions/workflows/docs.yml)
[![codecov](https://codecov.io/gh/unklstewy/BIG_SKIES_FRAMEWORK/branch/main/graph/badge.svg)](https://codecov.io/gh/unklstewy/BIG_SKIES_FRAMEWORK)
```

## Required Secrets

Configure these in your GitHub repository settings:

### Docker Hub
- `DOCKERHUB_USERNAME`: Docker Hub username
- `DOCKERHUB_TOKEN`: Docker Hub access token

### Code Coverage
- `CODECOV_TOKEN`: Codecov upload token

### SonarCloud (Optional)
- `SONAR_TOKEN`: SonarCloud analysis token

### Slack Notifications (Optional)
- `SLACK_WEBHOOK_URL`: Slack webhook for alerts

## Local Development

The CI/CD pipeline can be tested locally using:

```bash
# Run linting
make lint

# Run tests with coverage
make test

# Run integration tests
make test-integration

# Build Docker images
make docker-build
```

## Troubleshooting

### Common Issues

1. **Test Failures**: Check PostgreSQL/MQTT service logs in GitHub Actions
2. **Docker Build Failures**: Ensure Docker Buildx is available
3. **Security Scan Failures**: Review vulnerability reports and update dependencies
4. **Performance Regressions**: Check benchmark results in GitHub Actions artifacts

### Debugging Workflows

1. **View Logs**: Click on workflow run → View details
2. **Download Artifacts**: Check workflow artifacts for test results, coverage reports
3. **Re-run Failed Jobs**: Use GitHub Actions re-run functionality
4. **Local Testing**: Replicate CI environment using `make docker-up`

## Maintenance

### Regular Tasks

- **Weekly**: Review Dependabot PRs
- **Monthly**: Update GitHub Actions versions
- **Quarterly**: Review and update security scanning rules

### Performance Monitoring

- Monitor CI execution times
- Track test coverage trends
- Review performance benchmark results
- Check for flaky tests

## Integration with Development Workflow

The CI/CD pipeline integrates with the development workflow:

1. **Feature Development**: Push to feature branches triggers CI validation
2. **Pull Requests**: Automated testing and quality checks
3. **Merging**: Main branch deployment to staging
4. **Releases**: Tagged releases deploy to production

This ensures high code quality and reliable deployments throughout the development lifecycle.