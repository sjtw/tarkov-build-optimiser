---
name: data-access-patterns
description: Guidelines for database interactions using raw SQL and idiomatic Go patterns
---

# Data Access Patterns

This project follows a strict pattern for database access, prioritizing explicit SQL over ORMs for performance and clarity.

## Core Principles

- **No ORM**: Use `database/sql` directly.
- **Raw SQL**: Write explicit, readable SQL queries.
- **Idiomatic Naming**: Follow consistent naming for repository functions.
- **Transactions for Writes**: Always use `*sql.Tx` for operations that modify data.
- **Package Location**: Keep database models and access logic in `internal/models/`.

## Function Naming Conventions

| Pattern | Purpose | Example |
|---------|---------|---------|
| `Get[Entity]ById` | Retrieve a single entity by its primary key | `GetWeaponById(db *sql.DB, id string) (*Weapon, error)` |
| `Get[Entities]By[Field]` | Retrieve multiple entities by a specific field | `GetTraderOffersByItemID(db *sql.DB, itemID string) ([]TraderOffer, error)` |
| `Upsert[Entity]` | Insert or update a single entity | `UpsertWeapon(tx *sql.Tx, weapon Weapon) error` |
| `UpsertMany[Entity]` | Insert or update a slice of entities | `UpsertManyWeapon(tx *sql.Tx, weapons []Weapon) error` |
| `Purge[Entity]` | Delete all records of an entity | `PurgeOptimumBuilds(db *sql.DB) error` |

## Writing Queries

### Select with Joins and JSON
For complex nested objects, use PostgreSQL's JSON functions to retrieve structured data in a single query.

```go
query := `
    SELECT w.name,
           w.item_id,
           jsonb_agg(jsonb_build_object(
               'slot_id', ws.slot_id, 
               'name', ws.name
           )) as slots
    FROM weapons w
    JOIN slots ws ON w.item_id = ws.item_id
    WHERE w.item_id = $1
    GROUP BY w.name, w.item_id;`
```

### Upsert with ON CONFLICT
Always handle potential conflicts explicitly.

```go
query := `
    INSERT INTO weapons (item_id, name)
    VALUES ($1, $2)
    ON CONFLICT (item_id) DO UPDATE SET
        name = EXCLUDED.name;`
```

## Implementation Example

### Model Structs
Use JSON tags that match the database column names or API response keys.

```go
type Weapon struct {
    ID   string `json:"item_id"`
    Name string `json:"name"`
}
```

### Repository Function
Notice the use of `tx *sql.Tx` for writes and `db *sql.DB` for reads.

```go
func UpsertWeapon(tx *sql.Tx, weapon Weapon) error {
    query := `INSERT INTO weapons (item_id, name) VALUES ($1, $2) ON CONFLICT (item_id) DO UPDATE SET name = $2;`
    _, err := tx.Exec(query, weapon.ID, weapon.Name)
    return err
}
```

## Best Practices

- ✅ **Explicit Scanning**: Scan rows into structs carefully.
- ✅ **Deferred Closing**: Always `defer rows.Close()` after a query.
- ✅ **Error Handling**: Check errors after `rows.Scan()` and `rows.Err()`.
- ✅ **Use EXCLUDED**: In `ON CONFLICT` clauses, use `EXCLUDED.column_name` for clarity when updating.
- ❌ **Avoid SELECT ***: Always list the columns you need.
- ❌ **Avoid Global State**: Pass the `*sql.DB` or `*sql.Tx` as a parameter.
