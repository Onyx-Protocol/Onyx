## API

### Idempotency

Most of the API endpoints are idempotent. Some endpoints require a `client_token` parameter that is used for
ensuring idempotency. These client tokens are only used as an idempotency key, and cannot be used to lookup
entities later.
