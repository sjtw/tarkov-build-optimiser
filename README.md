# Tarkov Build Optimiser

A tool for calculating optimal weapon builds in Escape from Tarkov.

## Goals

- [x] calculate an optimum weapon build
- [ ] write a couple of tests to make sure optimum builds are actually optimum
- [ ] do step 1. but restricted by trader availability
- [ ] API for querying optimised builds
- [ ] create something to convert an `optimum_build` json into the structure used by totovbuilder.com import feature.
- [ ] provide cost effective alternatives
- [ ] allow a budget to be provided
- [ ] Make a UI
- [ ] Finish UI maybe

## Prerequisites

Required

- `docker`
- `docker-compose`
- `go`

Optional

Only used for updating the tarkovdev graphql schema.
- `nodejs`

## Development

### Tasks

building/running/starting dev infrastructure/etc. can all be handled using `Task`. See the task list;

```
task --list-all
```

Node.js can be used for updating the tarkovdev schema. If you need to do this, you can set up a local dev environment with:

```
task init
```

alternatively if you don't have node installed:

```
task init:go-only
```

This will install all required dependencies, set up a postgres docker container for development & apply all migrations.

#### Migrations
If you just want to run the migrations manually you can use the following. If it's your first time setting up the project, `task init` should've handled this for you.
```
task migrate:up
task migrate:down
```

#### Importer

The importer pulls all weapons and weapon mods from tarkov.dev & stores them in the `tarkov-build-optimiser` database.

```
task importer:start
```

The importer by default populates a json file cache of weapons & mods. To save hammering tarkov.dev each time you need to repopulate the database, you can repopulate it from the file cache using:

```
task importer:start:use-cache
```

If you only want to cache the results from tarkov.dev, without repopulating the db:
```
task importer:start:cache-only
```

#### Evaluator

Once the importer is finished, you should be able to run the evaluator. This creates up to two weapon builds for each weapon, with one prioritising ergonomics, the other prioritising recoil.

```
task evaluator:start
```

This can take a while as it exhaustively optimises each fork in the weapon builds tree. Once finished the `optimum_builds` table should be populated with the best builds for each individual weapon in the game.

#### Syncing tarkov.dev GraphQL API schema

The GraphQL queries used by the project are can be found in `internal/tarkovdev/schemas/queries.graphql`, the rest of the GraphQl client code is auto-generated. We use `graphql-inspector` to generate `schema.graphql` through an introspection query to tarkov.dev, then `genqlient` to generate golang functions & types for each of these queries in `generated-queries.go`. This can be done using:
```
task tarkovdev
```

If you only need to rebuild the `schema.graphql`;

```
task tarkovdev:get-schema
```

Or to only regenerate `generated-queries.go`;

```
task tarkovdev:regenerate
```

### Useful Links

https://api.tarkov.dev/api/graphql
