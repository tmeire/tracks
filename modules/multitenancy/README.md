# Multitenancy Module for Tracks

This module adds multitenancy support to the Tracks framework, allowing each tenant to have its own database while sharing the same application code.

## Features

- Each tenant has its own database
- A central database keeps track of available tenants
- A central database keeps track of users and their roles within a tenant
- Controllers only have access to the tenant-specific database
- Tenant resolution based on subdomain

## Usage

### Adding Multitenancy to Your Application

To add multitenancy support to your application, add the multitenancy module to your router setup:

```go
import (
    "github.com/tmeire/tracks"
    "github.com/tmeire/tracks/modules/multitenancy"
)

func main() {
    // Set up the central database
    centralDB, err := sqlite.New(filepath.Join(".", "data", "central.sqlite"))
    if err != nil {
        log.Fatalf("failed to connect to database: %v", err)
    }
    defer centralDB.Close()

    // Apply migrations to the central database
    err = db.MigrateUp(context.Background(), centralDB, db.CentralDatabase)
    if err != nil {
        log.Fatalf("failed to apply migrations: %v", err)
    }

    // Set up the router with multitenancy support
    tracks.New(centralDB).
        Module(multitenancy.WithMultitenancy(centralDB)).
        // Add other modules and resources
        Run()
}
```

### Updating Controllers to Use Tenant Databases

Controllers need to be updated to use the tenant database from the request context. Here's an example:

```go
import (
    "net/http"

    "github.com/tmeire/tracks/database"
    "github.com/tmeire/tracks/modules/multitenancy"
    "github.com/username/project/models"
)

type MyController struct {}

func (c *MyController) Index(r *http.Request) (any, error) {
    // Get the tenant database from the request context
    tenantDB := multitenancy.GetTenantDB(r.Context())

    // Create a repository using the tenant database
    repo := db.NewRepository[*models.MyModel](tenantDB)

    // Use the repository to query the tenant database
    return repo.FindAll()
}
```

### Creating Tenants

#### Using the CLI

The easiest way to create a new tenant is to use the CLI command:

```bash
# Create a new tenant with name and subdomain
go run internal/tracks/cli/main.go tenant create "Tenant Name" "subdomain"
```

This will:
1. Validate that the subdomain is not in the blacklist of reserved names
2. Create a new tenant record in the central database
3. Create a new database for the tenant
4. Apply all migrations to the tenant database

##### Subdomain Blacklist

For security reasons, certain common subdomain names are blacklisted and cannot be used when creating a new tenant. These include:

- admin
- api
- app
- blog
- dashboard
- dev
- docs
- help
- mail
- secure
- shop
- staging
- status
- support
- test
- www

Attempting to create a tenant with a blacklisted subdomain will result in an error message.

#### Programmatically

To create a new tenant programmatically, use the `TenantDB` instance:

```go
import (
    "context"

    "github.com/tmeire/tracks/modules/multitenancy"
)

func CreateTenant(ctx context.Context, tenantDB *multitenancy.TenantDB, name, subdomain string) (*multitenancy.Tenant, error) {
    return tenantDB.CreateTenant(ctx, name, subdomain)
}
```

### Managing User Roles

To manage user roles within a tenant, use the `UserRole` model and a repository:

```go
import (
    "context"

    "github.com/tmeire/tracks/database"
    "github.com/tmeire/tracks/modules/multitenancy"
)

func AssignUserToTenant(ctx context.Context, centralDB db.Database, userID, tenantID int64, role string) error {
    userRole := &multitenancy.UserRole{
        UserID:   userID,
        TenantID: tenantID,
        Role:     role,
    }

    repo := db.NewRepository[*multitenancy.UserRole](centralDB)
    _, err := repo.Create(userRole)
    return err
}
```

## Architecture

The multitenancy module consists of the following components:

1. **Models**: `Tenant` and `UserRole` models for storing tenant and user role information
2. **TenantDB**: Manages database connections for tenants
3. **Middleware**: Resolves the tenant from the request and sets up the appropriate database connection
4. **Module**: Registers the multitenancy functionality with the router

### Database Structure

#### Migration Structure

The project uses separate migration directories for central and tenant databases:

- `migrations/central/`: Contains migrations for the central database (tenants, user_roles, etc.)
- `migrations/tenant/`: Contains migrations for tenant databases (tenant-specific data)

When creating a new tenant, only the tenant migrations are applied to the tenant database. Central migrations are only applied to the central database.

#### Central Database Tables

The module uses two tables in the central database:

1. **tenants**: Stores information about available tenants
   - `id`: Auto-incrementing primary key
   - `name`: The name of the tenant
   - `subdomain`: The subdomain for the tenant (unique)
   - `db_path`: The path to the tenant's database file
   - `created_at`: Timestamp for when the tenant was created
   - `updated_at`: Timestamp for when the tenant was last updated

2. **user_roles**: Stores information about user roles within tenants
   - `id`: Auto-incrementing primary key
   - `user_id`: The ID of the user
   - `tenant_id`: The ID of the tenant
   - `role`: The role of the user within the tenant
   - `created_at`: Timestamp for when the user role was created
   - `updated_at`: Timestamp for when the user role was last updated

Each tenant has its own database file that contains all the tenant-specific data.
