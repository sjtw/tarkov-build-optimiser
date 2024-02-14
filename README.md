## Setup

- Install `docker` & `docker-compose`
- Install tooling:

```
./scripts/install-deps.sh
```

Optional:

- Install nodejs (if you need to update tarkovdev schema)

```
./scripts/install-node-deps.sh
```

## Development

### Tasks

building/running/starting dev infrastructure/etc can all be handled using `Task`. See the task list;

```
task --list-all
```

#### Running Locally

```
task start:local
```

#### Migrations

```
task migrate:up
task migrate:down
```
