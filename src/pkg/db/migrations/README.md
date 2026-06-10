# openocta.db migrations

Migration files are embedded by `pkg/db` and executed by `db.InitDB`.

## Naming

- Use `NNN_short_name.sql`.
- Versions are monotonically increasing positive integers.
- A version number must never be reused.

## Execution

- `schema_migrations` is created before embedded migrations run.
- Files run in ascending version order.
- Each migration runs inside one transaction.
- Applied migrations are recorded with `version`, `name`, `checksum`, and `applied_at`.
- Re-running startup is idempotent: already applied versions are skipped after checksum verification.

## Rollback policy

Rollbacks are forward-only. If a migration is wrong, add a new migration that repairs the schema or data. Do not edit an applied migration, because checksum verification will fail on existing installations.

## Ownership

All new `openocta.db` tables must be introduced here. Module-level `CREATE TABLE` calls are compatibility guards only and should be removed as repositories move to migration-managed schemas.
