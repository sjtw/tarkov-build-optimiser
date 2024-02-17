# Tarkov Build Optimiser

A tool for calculating optimal weapon builds in Escape from Tarkov.

## Goals

1. calculate an optimum weapon build
2. do step 1. but restricted by trader availability
3. provide cost effective alternatives
4. UI maybe

## Prerequisites

Install

- `docker`
- `docker-compose
- `go`

Optionally, install `nodejs`

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

#### Running Locally

```
task start:local
```

#### Migrations

```
task migrate:up
task migrate:down
```

#### Syncing tarkov.dev GraphQL API schema

retrieve latest schema & regenerate the api client package in `internal/tarkovdev`

```
task tarkovdev
```

retrieve schema only

```
task tarkovdev:get-schema
```

regenerate api client package only

```
task tarkovdev:regenerate
```

### Useful Links

https://api.tarkov.dev/api/graphql
