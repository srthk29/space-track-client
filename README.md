# Space-Track Client

A Go client for interacting with the Space-Track service with built-in authentication session management, cookie handling, request rate limiting, and structured request logging.

## Overview

The client is built around two components:

- `Client` manages authenticated HTTP requests to Space-Track.
- `Auth` manages the Space-Track login session, session refresh, and logout operations.

The implementation uses a shared `http.CookieJar` for both authentication requests and normal API requests. Authentication is ensured before every request made through `Client.Do`.

The current implementation targets:

```text
https://www.space-track.org
```

## Features

- Space-Track username/password authentication.
- Automatic login when no authentication session is available.
- Automatic session refresh before expiration.
- Automatic re-login when session refresh fails.
- Shared cookie jar between authentication and normal requests.
- Cookie expiration tracking.
- Request rate limiting.
- Structured logging with `log/slog`.
- Configurable log level through the `LOG_LEVEL` environment variable.
- Environment-based credential configuration.

## Requirements

The client uses the following Go packages:

- `github.com/joho/godotenv`
- `golang.org/x/net/publicsuffix`
- `golang.org/x/time/rate`

The standard library packages used include:

- `context`
- `encoding/json`
- `fmt`
- `log/slog`
- `net/http`
- `net/http/cookiejar`
- `os`
- `strings`
- `sync`
- `time`

## Configuration

The client loads environment variables using `godotenv.Load()`. A `.env` file can therefore be used during local development.

The following environment variables are supported:

| Variable | Description | Default |
| --- | --- | --- |
| `SPACETRACK_USERNAME` | Space-Track account identity/username | None |
| `SPACETRACK_PASSWORD` | Space-Track account password | None |
| `LOG_LEVEL` | Logging level: `DEBUG`, `INFO`, `WARN`, or `ERROR` | `INFO` |

Example `.env`:

```dotenv
SPACETRACK_USERNAME=your-username
SPACETRACK_PASSWORD=your-password
LOG_LEVEL=INFO
```

Credentials are read when `NewHttpClient()` is called.

## Creating a Client

Create a Space-Track client with:

```go
client, err := NewHttpClient()
if err != nil {
    return err
}
```

`NewHttpClient`:

1. Loads environment variables.
2. Creates a cookie jar using `publicsuffix.List`.
3. Creates a structured JSON logger.
4. Configures the authentication client.
5. Configures the main HTTP client.
6. Configures a rate limiter.
7. Returns a `Client`.

The Space-Track base URL is currently hard-coded as:

```text
https://www.space-track.org
```

## Making Requests

Authenticated requests should be sent through `Client.Do`:

```go
req, err := http.NewRequestWithContext(
    ctx,
    http.MethodGet,
    "https://www.space-track.org/...",
    nil,
)
if err != nil {
    return err
}

resp, err := client.Do(req)
if err != nil {
    return err
}
defer resp.Body.Close()
```

Before the request is sent, `Client.Do` calls `Auth.ensureAuth`.

This means authentication is handled transparently for requests made through the client.

## Authentication Lifecycle

The authentication flow is managed by the `Auth` type.

### Initial Login

If the client has no known session expiration time, `ensureAuth` calls `login`.

The login request is sent as:

```text
POST /ajaxauth/login
```

The request body contains:

```text
identity=<username>&password=<password>
```

with the content type:

```text
application/x-www-form-urlencoded
```

A successful login requires an HTTP `200 OK` response.

Cookies returned by the login response are inspected and their expiration time is used to track the session lifetime.

If no cookie expiration time is available, the client assumes a two-hour cookie lifetime.

### Session Refresh

When the known session is within 30 minutes of expiration, the client attempts to refresh the session by requesting:

```text
GET /app/data/whoami
```

The response is decoded into the `Lifetime` structure:

```go
type Lifetime struct {
    LoggedIn bool      `json:"logged_in"`
    Identity string    `json:"identity"`
    ExpireAt time.Time `json:"session_expiration"`
}
```

If the response indicates that the user is still logged in, the session expiration time is updated from `session_expiration`.

If the refresh fails, the client falls back to logging in again.

### Authentication Decision Flow

The authentication logic is effectively:

```text
No known expiration
    |
    v
Login
    |
    v
Session expiration known
    |
    +-- More than 30 minutes remaining --> Continue
    |
    +-- 30 minutes or less remaining --> Refresh
                                             |
                                             +-- Refresh succeeds --> Continue
                                             |
                                             +-- Refresh fails ------> Login
```

Authentication state changes are protected by a mutex so that authentication operations are serialized.

## Logout

The `Auth` type provides a logout operation that sends:

```text
GET /ajaxauth/logout
```

A successful logout requires an HTTP `200 OK` response.

The current `Client` API does not expose logout directly, so applications using this implementation would need to provide an appropriate wrapper if explicit logout is required.

## Cookie Management

The client creates one `http.CookieJar` and assigns it to both HTTP clients:

- The raw authentication client.
- The main Space-Track client.

This allows cookies established during authentication to be reused by subsequent requests.

The cookie jar is configured with:

```go
cookiejar.New(&cookiejar.Options{
    PublicSuffixList: publicsuffix.List,
})
```

The authentication client and main client share the same jar instance.

## Rate Limiting

The main HTTP client is configured with a rate limiter:

```go
rate.NewLimiter(rate.Every(time.Minute/5), 1)
```

This permits requests at a rate of approximately five requests per minute, with an initial burst capacity of one.

The rate limiter is applied through the transport chain used by the main client.

Authentication requests made through the raw authentication client do not use this rate limiter in the current implementation.

## HTTP Transport Chain

The main client configures its transport as:

```go
Chain(
    http.DefaultTransport,
    NewRateLimiter(limiter),
    NewLog(logger),
)
```

The authentication client configures its transport as:

```go
Chain(
    http.DefaultTransport,
    NewLog(logger),
)
```

The source currently contains a commented-out authentication transport:

```go
// NewAuth(auth),
```

Authentication is instead performed explicitly by `Client.Do` through:

```go
c.auth.ensureAuth(req.Context())
```

The implementations of `Chain`, `NewRateLimiter`, and `NewLog` are not included in the provided source files, so their exact behavior is determined by their respective implementations elsewhere in the project.

## Logging

The client uses Go's `log/slog` package with a JSON handler.

The logger is configured with:

```go
slog.NewJSONHandler(
    os.Stdout,
    &slog.HandlerOptions{
        AddSource: true,
        Level: getLogLevelEnv().Slog(),
    },
)
```

Logs are written to standard output.

The supported log levels are:

- `DEBUG`
- `INFO`
- `WARN`
- `ERROR`

Invalid or empty `LOG_LEVEL` values fall back to `INFO`.

Authentication-related operations emit debug logs containing authentication lifetime information and cookies.

The exact request and response logging behavior is implemented by `NewLog`, which is not included in the provided source files.

## Error Handling

The authentication methods return errors when:

- An HTTP request cannot be created.
- The HTTP request fails.
- Space-Track returns a non-`200 OK` response.
- A response body cannot be decoded.
- The authentication response indicates that the session is not logged in.

Examples of errors include:

```text
login failed: <status>
session extend failed: <status>
session expired
logout failed: <status>
```

`Client.Do` returns an authentication error without sending the requested HTTP request if `ensureAuth` fails.

## Example Usage

A typical request flow is:

```go
package main

import (
    "context"
    "fmt"
    "net/http"
)

func main() {
    client, err := NewHttpClient()
    if err != nil {
        panic(err)
    }

    req, err := http.NewRequestWithContext(
        context.Background(),
        http.MethodGet,
        "https://www.space-track.org/...",
        nil,
    )
    if err != nil {
        panic(err)
    }

    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    fmt.Println(resp.Status)
}
```

The first call to `client.Do` authenticates the session if necessary. Subsequent calls reuse the session cookie stored in the shared cookie jar.

## Client Architecture

The main components can be represented as:

```text
                         +------------------+
                         |     Client       |
                         +--------+---------+
                                  |
                                  | Do(req)
                                  v
                         +------------------+
                         |      Auth        |
                         |  ensureAuth()    |
                         +--------+---------+
                                  |
                    +-------------+-------------+
                    |                           |
                    v                           v
             Session valid                Session invalid
                    |                           |
                    |                    +------+------+
                    |                    |             |
                    |                    v             v
                    |                  login()     refresh()
                    |                    |             |
                    |                    +------+------+ 
                    |                           |
                    +-------------+-------------+
                                  |
                                  v
                         +------------------+
                         |  Shared Cookie   |
                         |       Jar        |
                         +--------+---------+
                                  |
                                  v
                         +------------------+
                         | Main HTTP Client |
                         | Rate Limiter     |
                         | Request Logging  |
                         +------------------+
```

## Project Scope

Based on the provided source files, this package currently provides the HTTP client and authentication/session infrastructure.

The provided implementation does not define higher-level methods for specific Space-Track API resources or query construction. Applications can build an `http.Request` for the required Space-Track endpoint and submit it through `Client.Do`.

The source files provided also do not define:

- Space-Track API resource-specific methods.
- Query builder APIs.
- Response models for Space-Track data.
- Pagination helpers.
- Retry policies.
- Persistent cookie storage.
- Persistent authentication state across process restarts.
- An exported logout method on `Client`.
- Implementations of `Chain`, `NewRateLimiter`, or `NewLog`.

These capabilities should be documented separately if they are implemented elsewhere in the project.

## Authentication and Persistence

The current implementation keeps the cookie jar in memory.

Because the cookie jar is created by `NewHttpClient()` and is not persisted, authentication cookies are lost when the application process exits. On the next application start, the client will need to authenticate again.

The current source therefore provides in-process session reuse, but not persistent authentication across service restarts.

## Concurrency

Authentication state is protected by a `sync.Mutex`.

`ensureAuth` acquires the mutex for the complete authentication check and any required login or refresh operation. This prevents multiple concurrent callers from simultaneously performing authentication state transitions.

The main HTTP client and its cookie jar are shared by the `Client` instance.

Applications should still follow the normal `net/http` rules for safely using a shared `http.Client` and should ensure that request and response bodies are handled correctly.

## License

No license information is provided in the supplied source files.

Add the project's applicable license information here before publishing the client as an open-source project.
