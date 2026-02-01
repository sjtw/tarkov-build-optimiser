---
name: api-endpoint-pattern
description: Standards for creating and organizing HTTP API endpoints using the Echo framework
---

# API Endpoint Pattern

This project uses the [Echo](https://echo.labstack.com/) web framework. Endpoints are organized by domain to keep the routing logic maintainable.

## Router Structure

### 1. Main Router (`internal/router/router.go`)
The main router initializes Echo, sets up global middleware, and groups routes by domain.

```go
func NewRouter(config Config) *echo.Echo {
    e := echo.New()
    e.Use(middleware.Logger())
    
    api := e.Group("/api")
    itemsrouter.Bind(api.Group("/items"), config.DB.Conn)
    
    return e
}
```

### 2. Sub-Routers (`internal/router/[domain]/router.go`)
Each domain (e.g., `items`) has its own sub-router with a `Bind` function. This function takes an `*echo.Group` and the database connection.

```go
func Bind(e *echo.Group, db *sql.DB) *echo.Group {
    e.GET("/list", func(c echo.Context) error {
        // ... handler logic
    })
    return e
}
```

## Handler Implementation

### Parameter Parsing
Helper functions should be used to parse complex parameters like trader levels or pagination.

```go
func getTraderLevelParams(c echo.Context) ([]models.TraderLevel, error) {
    // ... logic to parse query params
}
```

### Dependency Injection
Pass the database connection or other dependencies into the `Bind` function and then into the handlers.

### Response Handling
- **Success**: Use `c.JSON(200, data)` for structured data or `c.String(200, "OK")` for simple responses.
- **Error**: Use `c.String(code, err.Error())` for errors. Log significant errors using `zerolog`.

## Best Practices

- ✅ **Grouped Routes**: Always use `e.Group()` for logical separation.
- ✅ **Domain Separation**: Keep sub-routers in their own packages within `internal/router/`.
- ✅ **Explicit DB**: Pass `*sql.DB` to handlers rather than using a global instance.
- ✅ **Consistent Naming**: Use plural names for collections (e.g., `/items`, `/weapons`).
- ✅ **Query Params**: Use query parameters for filtering and optional configuration.
- ❌ **Bloated Handlers**: Move complex business logic into the `internal/models` or `internal/evaluator` packages.
- ❌ **Magic Numbers**: Use HTTP constants for status codes if preferred, but be consistent.

## Adding a New Endpoint

1. Create a new directory in `internal/router/` if a new domain is introduced.
2. Implement the `Bind` function in `internal/router/[domain]/router.go`.
3. Add the domain group and call the `Bind` function in `internal/router/router.go`.
4. Implement the necessary data access logic in `internal/models/`.
