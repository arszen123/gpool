# Simple resource pool

## Configuration

- `Max` - Maximum number of resources the pool can have. (default=0)
- `AcquireTimeoutMillis` - Maximum duration before timing out a resource acquire. Returns a `ErrorAcquireTimeout` error, if exceeded. (default=0, unlimited)
- `MaxWaitingClients` - Maximum number of queued requests allowed. Additional `Acquire` calls will retrun an `ErrorMaximumWaitingClientsExceeded` error. (default=0, unlimited)
- `Factory`
  - `Create` - A function that the pool will call to create a new resource.
  - `Destroy` - When provided, the `Destroy` function is called when a resource is about to be destroyed.
  - `Validate` - When provided, the `Validate` function is called before retrieving a resource to validate whether it's still active.

## TODO

- Update tests
- Cleanup codebase
- Implement idle resources removal