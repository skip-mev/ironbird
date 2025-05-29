# Database Migration: PostgreSQL to SQLite

This document describes the migration from PostgreSQL to SQLite for the Ironbird project.

## Overview

The database has been migrated from PostgreSQL to SQLite while maintaining the same schema structure and API compatibility. This change provides several benefits:

- **Simplified deployment**: No need for a separate PostgreSQL server
- **Reduced dependencies**: SQLite is embedded and requires no external setup
- **Better portability**: Database is a single file that can be easily backed up and moved
- **Improved development experience**: No need to set up PostgreSQL for local development

## Changes Made

### 1. Database Implementation

- **File**: `db/postgres.go` - Replaced PostgreSQL implementation with SQLite
- **New Functions**: 
  - `NewSQLiteDB(dbPath string)` - Creates a new SQLite database connection
  - `NewPostgresDB(dbPath string)` - Backward compatibility alias for `NewSQLiteDB`
- **Type Aliases**: `PostgresDB = SQLiteDB` for backward compatibility

### 2. Migration Files

- **Updated**: `migrations/001_create_workflows_table.up.sql` - SQLite-compatible schema
- **Updated**: `migrations/001_create_workflows_table.down.sql` - SQLite-compatible cleanup

### 3. Schema Changes

The schema has been adapted for SQLite while maintaining the same logical structure:

```sql
-- PostgreSQL (old)
id SERIAL PRIMARY KEY
nodes JSONB DEFAULT '[]'::jsonb
created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()

-- SQLite (new)  
id INTEGER PRIMARY KEY AUTOINCREMENT
nodes TEXT DEFAULT '[]'
created_at DATETIME DEFAULT CURRENT_TIMESTAMP
```

### 4. Dependencies

- **Added**: `github.com/mattn/go-sqlite3` - SQLite driver
- **Added**: `github.com/golang-migrate/migrate/v4/database/sqlite3` - Migration support
- **Removed**: PostgreSQL-specific dependencies are no longer needed

## Usage

### Creating a Database Connection

```go
// New way (recommended)
db, err := db.NewSQLiteDB("./ironbird.db")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Old way (still works for backward compatibility)
db, err := db.NewPostgresDB("./ironbird.db")
if err != nil {
    log.Fatal(err)
}
defer db.Close()
```

### Running Migrations

```go
err := db.RunMigrations("./migrations")
if err != nil {
    log.Fatal(err)
}
```

### Database File Location

The SQLite database is stored as a single file. Common locations:

- **Development**: `./ironbird.db`
- **Production**: `/var/lib/ironbird/ironbird.db`
- **Testing**: `/tmp/test_ironbird.db`

## Backward Compatibility

All existing code that uses the database interface will continue to work without changes:

- `db.DB` interface remains unchanged
- All CRUD operations have the same signatures
- `NewPostgresDB()` function still exists (now creates SQLite connection)
- `PostgresDB` type still exists (now alias for `SQLiteDB`)

## Performance Considerations

SQLite provides excellent performance for the expected workload:

- **Reads**: Very fast, especially with WAL mode enabled
- **Writes**: Good performance for moderate write loads
- **Concurrency**: Supports multiple readers, single writer
- **Indexes**: All existing indexes are preserved and optimized for SQLite

## Configuration

SQLite is configured with optimal settings:

```sql
PRAGMA foreign_keys = ON;     -- Enable foreign key constraints
PRAGMA journal_mode = WAL;    -- Enable Write-Ahead Logging for better concurrency
```

## Backup and Recovery

### Backup
```bash
# Simple file copy (when database is not in use)
cp ironbird.db ironbird_backup.db

# Online backup using SQLite command
sqlite3 ironbird.db ".backup ironbird_backup.db"
```

### Recovery
```bash
# Restore from backup
cp ironbird_backup.db ironbird.db
```

## Testing

Run the database tests to verify functionality:

```bash
cd db
go test -v
```

The tests verify:
- Basic CRUD operations
- Migration functionality
- Backward compatibility
- JSON field handling

## Migration from Existing PostgreSQL

If you have existing PostgreSQL data, you can migrate it using:

1. Export data from PostgreSQL:
```bash
pg_dump --data-only --inserts your_db > data.sql
```

2. Convert PostgreSQL syntax to SQLite (manual process for complex cases)

3. Import into SQLite:
```bash
sqlite3 ironbird.db < converted_data.sql
```

For this project, since the schema is simple, you can also recreate the data using the application's API.

## Troubleshooting

### Common Issues

1. **File permissions**: Ensure the application has read/write access to the database file and directory
2. **Concurrent access**: SQLite handles multiple readers but only one writer at a time
3. **File locking**: On some systems, ensure no other processes are accessing the database file

### Debugging

Enable SQLite debugging by setting environment variables:
```bash
export CGO_ENABLED=1
export SQLITE_DEBUG=1
```

## Future Considerations

- **Scaling**: If write concurrency becomes an issue, consider connection pooling or database sharding
- **Replication**: For high availability, implement file-based replication or consider moving back to PostgreSQL
- **Analytics**: For complex analytical queries, consider read replicas or data export to analytical databases
