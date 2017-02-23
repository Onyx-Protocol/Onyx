# Operations guide

### Topics

- [Monitoring and health checks](#monitoring-and-health-checks)

## Monitoring and health checks

cored exposes an authenticated health check endpoint, `/health`, via HTTP. The endpoint returns an empty 200 OK response.

For uptime monitoring, you can periodically check this endpoint. If your request returns anything else, then the server is unavailable.
