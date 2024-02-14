### Migrations

### Setup

- Install `docker` & `docker-compose`
- Install tooling:

```
./scripts/install-deps.sh
```

### Development

Everything should be handled through `Task`. These can be listed as such & should include meaningful descriptions:

```
task --list-all
```

#### Running DB & API Locally

```
task local:start
```
