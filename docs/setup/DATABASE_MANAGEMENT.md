# Database Management Guide

This guide covers database backup, restore, and persistence for the BIG SKIES Framework.

## Overview

The framework uses PostgreSQL 15 with:
- **Named volume**: `postgres_data` for data persistence
- **Bootstrap migrations**: Schema managed by bootstrap coordinator
- **No init scripts**: Database persists across container rebuilds

## Database Persistence

### How It Works

1. **First startup**: 
   - PostgreSQL container creates empty database
   - Bootstrap coordinator runs migrations to create schema
   - Data persists in `postgres_data` Docker volume

2. **Subsequent startups**:
   - Database loads from `postgres_data` volume
   - Bootstrap coordinator checks migrations (idempotent)
   - **Data is preserved** across `docker-compose down`/`up`

3. **Rebuilds**:
   - `make docker-build` only rebuilds **images**, not volumes
   - Database data remains intact
   - No reinitialization unless volume is deleted

### When Database IS Reinitialized

Database is **only** reset when:
- Volume is deleted: `docker volume rm bigskies_postgres_data`
- Using `docker-compose down -v` (removes volumes)
- Using `make docker-purge` (removes everything)

### When Database IS NOT Reinitialized

Database persists through:
- ✅ `make docker-build` (rebuilds images only)
- ✅ `make docker-up` / `make docker-down` (starts/stops containers)
- ✅ `make docker-restart` (restarts containers)
- ✅ Container crashes or restarts
- ✅ Code changes and rebuilds

## Backup Database

### Quick Backup

```bash
make db-backup
```

This creates:
- Timestamped SQL dump: `backups/database/bigskies_backup_YYYYMMDD_HHMMSS.sql.gz`
- Includes schema and data
- Compressed with gzip

### Backup Output

```
BIG SKIES Framework - Database Backup
======================================

Backing up database: bigskies
Container: bigskies-postgres
Output: backups/database/bigskies_backup_20260116_224500.sql

✅ Backup completed successfully!
   File: bigskies_backup_20260116_224500.sql
   Size: 2.3M
   Path: backups/database/bigskies_backup_20260116_224500.sql

Compressing backup...
✅ Compressed: bigskies_backup_20260116_224500.sql.gz (384K)

Recent backups:
total 768
-rw-r--r--  1 user  staff   384K Jan 16 22:45 bigskies_backup_20260116_224500.sql.gz
-rw-r--r--  1 user  staff   380K Jan 15 18:30 bigskies_backup_20260115_183000.sql.gz

To restore this backup:
  ./scripts/db-restore.sh backups/database/bigskies_backup_20260116_224500.sql.gz
```

### Manual Backup (if needed)

```bash
docker exec bigskies-postgres pg_dump -U bigskies bigskies | gzip > my_backup.sql.gz
```

## Restore Database

### Interactive Restore

```bash
make db-restore
```

This will:
1. Show available backups
2. Prompt for backup file to restore
3. Stop coordinators to prevent conflicts
4. Restore database
5. Restart coordinators

### Direct Restore

```bash
./scripts/db-restore.sh backups/database/bigskies_backup_20260116_224500.sql.gz
```

### Restore Output

```
BIG SKIES Framework - Database Restore
=======================================

⚠️  WARNING: This will replace the current database!

Backup file: backups/database/bigskies_backup_20260116_224500.sql.gz
Database: bigskies
Container: bigskies-postgres

Continue? Type 'yes' to proceed: yes

Stopping coordinators to prevent database conflicts...
bigskies-bootstrap
bigskies-datastore
[...]

Restoring database...
Decompressing backup...
✅ Database restored successfully!

Restarting coordinators...
✅ Coordinators restarted!

Verify with:
  docker-compose ps
  docker logs -f bigskies-bootstrap
```

## Database Status

Check database health and info:

```bash
make db-status
```

Output:
```
Database Status
===============

Container: bigskies-postgres
  Status: Up 2 hours

Connection Info:
  Host: localhost
  Port: 5432
  Database: bigskies
  User: bigskies

Database Size:
  Total:  12 MB

Tables:
                         List of relations
 Schema |            Name             | Type  |  Owner
--------+-----------------------------+-------+---------
 public | coordinator_config          | table | bigskies
 public | coordinator_config_history  | table | bigskies
 public | users                       | table | bigskies
 public | roles                       | table | bigskies
 [...]
```

## Common Scenarios

### Scenario 1: Before Risky Changes

```bash
# Backup before migrations or schema changes
make db-backup

# Make changes
docker restart bigskies-bootstrap

# If something breaks, restore
make db-restore
```

### Scenario 2: Fresh Start (Keep Data)

```bash
# Backup current state
make db-backup

# Stop everything
make docker-down

# Rebuild images
make docker-build

# Start (database persists automatically)
make docker-up

# Verify data is intact
make db-status
```

### Scenario 3: Complete Reset

```bash
# Optional: backup first
make db-backup

# Purge everything
make docker-purge

# Start fresh
./scripts/update-pgpass.sh
make docker-build
make docker-up

# Optionally restore backup
make db-restore
```

### Scenario 4: Test with Production Data

```bash
# On production: backup
make db-backup

# Copy backup to development machine
scp prod:/path/bigskies_backup_*.sql.gz ./backups/database/

# On development: restore
make db-restore
# Select the production backup
```

### Scenario 5: Database Corruption

```bash
# Stop coordinators
docker-compose stop

# Try to backup (might fail)
make db-backup

# If backup fails, restore from last known good
make docker-down
docker volume rm bigskies_postgres_data
make docker-up

# Wait for bootstrap to run migrations
docker logs -f bigskies-bootstrap

# Restore from backup
make db-restore
```

## Automated Backups

### Cron Job (Linux/Mac)

```bash
# Edit crontab
crontab -e

# Add daily backup at 2 AM
0 2 * * * cd /path/to/BIG_SKIES_FRAMEWORK && make db-backup

# Or with script
0 2 * * * /path/to/BIG_SKIES_FRAMEWORK/scripts/db-backup.sh
```

### Backup Retention Script

Create `scripts/backup-cleanup.sh`:
```bash
#!/bin/bash
# Keep last 7 daily backups, last 4 weekly backups

BACKUP_DIR="/path/to/backups/database"

# Keep last 7 daily
find "$BACKUP_DIR" -name "*.sql.gz" -mtime +7 -delete

# Weekly backups: copy Sunday backups to weekly/
# (implementation left as exercise)
```

## Direct Database Access

### psql Shell

```bash
docker exec -it bigskies-postgres psql -U bigskies -d bigskies
```

Inside psql:
```sql
-- List tables
\dt

-- Show table structure
\d coordinator_config

-- Query data
SELECT * FROM coordinator_config;

-- Exit
\q
```

### Execute SQL Command

```bash
docker exec bigskies-postgres psql -U bigskies -d bigskies -c "SELECT count(*) FROM users;"
```

### Execute SQL File

```bash
cat my_script.sql | docker exec -i bigskies-postgres psql -U bigskies -d bigskies
```

## Migrations vs Backups

### Migrations (Schema Changes)

- Managed by bootstrap coordinator
- Tracked in `schema_migrations` table
- Idempotent (safe to run multiple times)
- Version-controlled SQL files in `configs/sql/`
- Run automatically on bootstrap startup

### Backups (Data Preservation)

- Full database dumps (schema + data)
- Timestamped files in `backups/database/`
- NOT version-controlled (excluded by .gitignore)
- Created manually or via cron
- Used for disaster recovery, testing, data migration

## Troubleshooting

### "Database already exists" Error

This is **normal** on restarts. Bootstrap coordinator migrations are idempotent.

### Database Won't Start

```bash
# Check logs
docker logs bigskies-postgres

# Common fixes
make docker-down
docker volume rm bigskies_postgres_data
make docker-up
```

### Backup Fails

```bash
# Check container is running
docker ps | grep postgres

# Check disk space
df -h

# Try manual backup
docker exec bigskies-postgres pg_dump -U bigskies bigskies > manual_backup.sql
```

### Restore Fails

```bash
# Check backup file integrity
gunzip -t backups/database/bigskies_backup_*.sql.gz

# Try restoring to new database
docker exec bigskies-postgres createdb -U bigskies bigskies_test
gunzip -c backup.sql.gz | docker exec -i bigskies-postgres psql -U bigskies -d bigskies_test
```

### Volume Not Persisting

```bash
# Check volume exists
docker volume ls | grep bigskies

# Inspect volume
docker volume inspect bigskies_postgres_data

# Ensure docker-compose.yml uses named volume (not bind mount)
grep "postgres_data:" docker-compose.yml
```

## Production Recommendations

1. **Automated Backups**: Set up cron job for daily backups
2. **Backup Rotation**: Keep 7 daily, 4 weekly, 12 monthly
3. **Offsite Storage**: Copy backups to S3/GCS/remote server
4. **Test Restores**: Verify backups monthly by restoring to dev
5. **Monitor Backup Size**: Alert if size changes unexpectedly
6. **Backup Before Updates**: Always backup before schema changes
7. **Separate Credentials**: Use different passwords per environment
8. **Enable WAL Archiving**: For point-in-time recovery (advanced)

## References

- [PostgreSQL Backup Documentation](https://www.postgresql.org/docs/15/backup.html)
- [pg_dump Reference](https://www.postgresql.org/docs/15/app-pgdump.html)
- [Docker Volumes](https://docs.docker.com/storage/volumes/)
- Bootstrap Setup: `docs/setup/BOOTSTRAP_SETUP.md`

---

**Remember**: Always backup before `make docker-purge`! 💾
