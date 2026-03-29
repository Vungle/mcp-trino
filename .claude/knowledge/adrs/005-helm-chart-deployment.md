# ADR 005: Kubernetes deployment via Helm chart with configurable security

## Status
Accepted

## Context
mcp-trino needs to run in Kubernetes clusters alongside Trino. Kubernetes deployment requires configurable Trino connection parameters, OAuth credentials, allowlists, resource limits, health checks, and network policies. A standardized deployment mechanism was needed that works across development and production environments.

## Decision
A Helm chart in `charts/mcp-trino/` provides Kubernetes deployment with three value files: `values.yaml` (defaults), `values-development.yaml`, and `values-production.yaml`.

Key configuration areas in the Helm chart:
- **Trino connection:** Host, port, user, password, catalog, schema, SSL settings, query timeout
- **OAuth:** Enabled flag, mode, provider, JWT secret, OIDC configuration (issuer, audience, client ID/secret)
- **Allowlists:** Catalogs, schemas, tables arrays mapped to env vars
- **Security context:** `runAsNonRoot: true`, `readOnlyRootFilesystem: true`, `allowPrivilegeEscalation: false`, drop ALL capabilities, run as user 65534 (nobody)
- **Health checks:** Startup (10s initial, 6 retries), liveness (30s initial), readiness (5s initial) — all hitting `/status` endpoint
- **Resource limits:** 500m CPU / 512Mi memory limits, 100m CPU / 128Mi memory requests
- **Networking:** ClusterIP service, optional ingress with ALB annotations, optional NetworkPolicy for ingress/egress control
- **Scaling:** Optional HPA with CPU/memory targets, PodDisruptionBudget

Sensitive values (Trino password, JWT secret, OIDC client secret) are stored in Kubernetes Secrets via `templates/secret.yaml`. Non-sensitive configuration uses ConfigMap via `templates/configmap.yaml`.

**Code references:**
- Chart definition: `charts/mcp-trino/Chart.yaml`
- Default values: `charts/mcp-trino/values.yaml`
- Environment values: `charts/mcp-trino/values-development.yaml`, `charts/mcp-trino/values-production.yaml`
- Templates: `charts/mcp-trino/templates/` — deployment.yaml, service.yaml, configmap.yaml, secret.yaml, ingress.yaml, hpa.yaml, pdb.yaml, rbac.yaml, serviceaccount.yaml, networkpolicy.yaml
- Docker image: `Dockerfile` — multi-stage build, published to `ghcr.io/tuannvm/mcp-trino`

## Consequences
- **Positive:** Helm provides a standardized, repeatable deployment mechanism with environment-specific overrides.
- **Positive:** Security context defaults follow Kubernetes security best practices (non-root, read-only filesystem, no privilege escalation).
- **Positive:** Health check probes ensure Kubernetes restarts unhealthy pods and removes unready pods from service endpoints.
- **Positive:** NetworkPolicy support allows restricting traffic to/from the mcp-trino pods (Trino port 8080, HTTPS 443, DNS 53).
- **Negative:** Helm chart maintenance requires keeping templates in sync with new environment variables and configuration options.
- **Negative:** Three value files (default, development, production) can drift — changes must be reviewed across all files.

## Alternatives Considered
- **Plain Kubernetes manifests:** Raw YAML files without Helm. Rejected because it lacks templating for environment-specific configuration and makes upgrades harder to manage.
- **Kustomize:** Overlay-based approach. Rejected because the team already uses Helm across other services and consistency is preferred.
- **Docker Compose for production:** Extend the existing docker-compose.yml. Rejected because Docker Compose is not suitable for production Kubernetes environments — it lacks health checks, rolling updates, and auto-scaling.
