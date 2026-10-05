# Tracks CLI

The `tracks` CLI provides utilities for scaffolding, code generation, asset compilation, and database migrations for Tracks applications.

## Building the CLI

To compile the CLI binary:

```bash
go build -o tracks ./cli/main.go
```

## Available Commands

### `init`

Initializes a new Tracks application with the standard directory structure, configuration, layout, default controller, and Docker/OTel setup.

```bash
./tracks init <module_path>
# Example:
./tracks init github.com/myorg/myapp
```

### `generate`

Generates controllers and resourceful CRUD scaffolding.

#### Controller & Action:

```bash
./tracks generate controller <method> <path>
# Example:
./tracks generate controller GET /about
```
This generates `controllers/about.go`, `views/default/about.gohtml`, and registers the route in `main.go`.

#### Resource Scaffolding:

```bash
./tracks generate resource <name>
# Example:
./tracks generate resource posts
```
This generates:
- Controller implementing `tracks.Resource` (`controllers/posts.go`)
- Database model (`models/posts.go`)
- Views: `views/posts/index.gohtml`, `new.gohtml`, `show.gohtml`, `edit.gohtml`
- Route registration in `main.go`

### `db`

Manages database migrations via Goose. Note the nested subcommand syntax:

```bash
# Apply pending migrations
./tracks db db up [--type central|tenant] [--db path]

# Rollback migrations
./tracks db db down [--type central|tenant] [--db path]

# View migration status
./tracks db db status [--type central|tenant] [--db path]
```

### `assets`

Pre-processes and hashes static assets for production:

```bash
./tracks assets compile [-r|--remove-original]
```
This hashes assets in `public/` using MD5 and rewrites references.

### `version`

Prints the current version of Tracks:

```bash
./tracks version
```
