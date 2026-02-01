---
name: data-import-workflow
description: Managing the lifecycle of weapon and mod data from external sources to the local database
---

# Data Import Workflow

The importer fetches weapon and mod data from the [tarkov.dev](https://tarkov.dev) GraphQL API and populates the local database.

## When to Use

- After a database reset or fresh installation.
- When new items are added to Escape from Tarkov.
- When item properties (recoil, ergonomics, price) change.

## Commands

| Command | Description |
|---------|-------------|
| `task importer:start` | Fetches fresh data from the API and updates the database. |
| `task importer:start:use-cache` | Uses locally cached JSON files instead of calling the API. |
| `task importer:start:cache-only` | Fetches data from the API and updates the local JSON cache, but doesn't touch the database. |

## Workflow Steps

1. **Environment Setup**: Ensure your `.env` is configured and the database is running.
2. **Purge Existing Data**: The importer automatically purges existing weapons, mods, and trader offers to ensure a clean state.
3. **Fetch Data**: Data is either fetched via GraphQL or loaded from `file-caches/*.json`.
4. **Transform**: The external schema from tarkov.dev is mapped to our internal `models` structs.
5. **Persist**: Data is stored in the database using transactions.
6. **Invalidate Build Cache**: After import, all `optimum_builds` are purged because they are now stale.

## Extending the Importer

If you need to import new fields or types:

1. **Update GraphQL Query**: Modify `internal/tarkovdev/schemas/queries.graphql`.
2. **Regenerate Client**: Run `task tarkovdev:regenerate`.
3. **Update Models**: Add the new fields to the relevant struct in `internal/models/`.
4. **Update Mapping**: Update the conversion logic in `internal/importers/`.
5. **Update SQL**: Update the `Upsert` functions in `internal/models/` to include the new columns.

## Troubleshooting

- **Rate Limiting**: If you hit rate limits on the tarkov.dev API, use `--use-cache` for subsequent runs.
- **Mapping Errors**: If the GraphQL schema changes, you may need to run `task tarkovdev:get-schema` followed by `task tarkovdev:regenerate`.
- **Database Connection**: Ensure the importer can reach the database (check `POSTGRES_HOST` in `.env`).
