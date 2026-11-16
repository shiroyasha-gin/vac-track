# Vaccine Inventory

A desktop-first vaccine stock and usage tracker for a single clinic workstation, plus a separate public service that publishes reviewed monthly aggregates.

- **Desktop app:** Go, Wails v2, SQLite, sqlc, React, TypeScript, Tailwind CSS.
- **Local data:** one SQLite database in the operating system's per-user data directory.
- **Public output:** a separate HTTPS service that accepts only allow-listed monthly aggregate snapshots.
- **Local login:** bcrypt-hashed operator accounts gate the desktop app; the session is memory-only and auto-locks after 20 minutes of inactivity.

The application is an inventory and planning tool. It is not a medical device, a clinical decision system, or a substitute for official immunization records.

---

## Privacy boundary

The private database and the public service are deliberately separate. The desktop app never uploads the SQLite file.

| Private desktop data | Public aggregate snapshot |
| --- | --- |
| Patient names and optional patient codes | No patient data |
| Individual usage events, dates and notes | Monthly totals only |
| Vaccine catalog details and stock adjustments | Vaccine name, code, unit and monthly dose counts |
| Audit history and publication history | Reporting month, snapshot id and generation time |

The public payload is defined by `internal/publishing.Snapshot`. It contains:

```json
{
  "schemaVersion": 1,
  "snapshotId": "snap_…",
  "month": "2026-10",
  "aggregation": "monthly",
  "generatedAt": "2026-11-01T08:00:00Z",
  "totals": {
    "vaccineCount": 1,
    "openingBalance": 10,
    "stockReceived": 5,
    "totalUsage": 3,
    "remaining": 12
  },
  "vaccines": [
    {
      "name": "Seasonal influenza",
      "code": "FLU",
      "unit": "dose",
      "openingBalance": 10,
      "stockReceived": 5,
      "totalUsage": 3,
      "remaining": 12
    }
  ]
}
```

The publication backend builds this payload from aggregate fields only, recomputes all totals, and refuses to publish if the content changed after the operator reviewed the preview.

---

## Safe defaults and assumptions

These are the intended operating assumptions. They are not enforced as multi-user security controls.

- **Whole doses only.** Quantities are integers; fractional doses are rejected.
- **Balance rule:** `opening + received − active usage = remaining`. A transaction that would make stock negative is rejected.
- **Explicit monthly initialization.** Usage cannot be recorded until a monthly inventory row exists for that vaccine, even if opening and received stock are both zero.
- **One workstation, local accounts.** Admins manage users and station-wide settings. Staff accounts handle daily stock, patient and appointment work. There is no remote access or synchronization, and every account on this workstation sees the same local data.
- **No cloud storage.** The database, logs, backups and exports stay on the local machine unless the operator moves them.
- **Monthly public reports only.** The public service rejects anything other than a monthly aggregate snapshot.
- **Low-stock threshold defaults to 10 doses.** Change it in **Settings**.
- **UTC dates.** Calendar dates are stored as `YYYY-MM-DD`, months as `YYYY-MM`, and timestamps as RFC 3339 UTC.
- **Closed months are locked.** Routine stock and usage edits are rejected until the month is reopened with a reason; closing, reopening and corrections are audited.
- **No real patient data or secrets in source control.** Use a test database or the development data-directory override when developing.

### Data protection limits

The database directory is created with `0700` permissions and the database, log, backups and exports with `0600`. There is no application-level encryption. Protect the workstation account and use full-disk encryption where patient data is stored. Backups and patient-level CSV exports are copies of sensitive data and must be handled accordingly.

---

## Features

- Local bcrypt-hashed operator login with an in-memory session and a first-run default account.
- Password change from **Settings → أمان الحساب** after verifying the current password.
- Admin-only account management in **Settings → إدارة المستخدمين**: create users, edit names and roles, reset passwords, and deactivate accounts. The last active admin cannot be removed, and you cannot deactivate yourself.
- Failed-login throttling after five attempts, with a short block that grows on repeated failures.
- Audit events record the acting account (`actor_id`, `actor_username`) for every mutation.
- Vaccine catalog with archive support and optional codes, categories and units.
- Patient directory with a required name and optional local patient code.
- Patient file with complete dose history, per-vaccine coverage and latest dose dates, plus scheduled and completed appointments.
- Appointment registration with immediate stock deduction: recording a visit creates exactly one audited usage entry per vaccine, deducts its doses from that month's stock and appears in the monthly usage totals. Editing an appointment adjusts or voids the linked entries so stock never diverges. Cancelling an appointment restores the reserved doses.
- Monthly inventory table with numbered day columns (28–31 days), opening leftover, received stock, total usage and remaining stock.
- Carryover suggestion from the previous month, with a required reason when overridden.
- Audited stock adjustments, usage corrections and voiding instead of deletion.
- Month close and reopen workflow with reasons.
- Local monthly reports, daily usage trend, comparison with the previous month and a transparent demand forecast.
- A4 print and PDF export for the monthly report and for each vaccine report, with print-only headers, filters and page-safe tables.
- Aggregate-only CSV export and confidential patient-level CSV export.
- Verified SQLite backups and staged restore with automatic rollback.
- Aggregate snapshot API served by the separate public service; the desktop publish and audit screens are not part of this build.
- Public statistics site with loading, empty and error states.
- Local audit trail for every stock, usage, appointment and publication change.

---

## Prerequisites

### All platforms

- Go **1.26 or newer**. `go.mod` pins the patched `go1.27.3` toolchain, which Go downloads automatically when needed.
- Node.js **20 or newer** and npm.
- Wails CLI **v2.17.1 or newer**. Older Wails releases embed an `x/tools` version that cannot type-check Go 1.27 modules, which breaks binding generation.

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.1
wails doctor
```

If `wails` is not on `PATH`, add `$(go env GOPATH)/bin` to it.

### Linux

Wails needs GTK 3 and a WebKitGTK development package. On Debian/Ubuntu:

```sh
sudo apt install build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev
```

On Fedora:

```sh
sudo dnf install gcc pkg-config gtk3-devel webkit2gtk4.1-devel
```

This development machine has WebKitGTK 4.1 but not 4.0, so every Wails command uses the `webkit2_41` build tag:

```sh
wails build -tags webkit2_41
```

If your distribution provides WebKitGTK 4.0 instead, omit the tag. Check with `wails doctor`.

### macOS and Windows

Use the Wails prerequisites for your platform. The `webkit2_41` tag is Linux-specific. Windows builds can be cross-compiled from Linux with MinGW-w64. Windows is portable: keep `VacTrack.exe` in a writable folder and the database, backups and log stay beside it. macOS still needs a build on macOS.

---

## Quick start

```sh
# Install frontend dependencies once.
cd frontend
npm ci
cd ..

# Live development (starts Vite and the desktop window).
wails dev -tags webkit2_41
```

Production build:

```sh
wails build -tags webkit2_41 -nopackage
./build/bin/VacTrack
```

Remove `-nopackage` if you want the platform packaging step. On Linux, the plain binary is written to `build/bin/VacTrack`.

### Where data lives

The app resolves its data directory as follows:

| Platform | Default |
| --- | --- |
| Linux | `$XDG_DATA_HOME/vaccineshit`, or `~/.local/share/vaccineshit` |
| macOS | `~/Library/Application Support/VaccineInventory` |
| Windows | Folder that contains `VacTrack.exe`; `%LOCALAPPDATA%\VaccineInventory` only when that folder is read-only |

The directory contains:

```text
vaccineshit.db          # SQLite database (WAL mode)
vaccineshit.db-wal      # present while the database is open
vaccineshit.db-shm
application.log         # sanitized technical errors
backups/                # default backup directory and pre-restore copies
```

Override the location for development or testing:

```sh
VACCINESHIT_DATA_DIR=/tmp/vaccineshit-dev wails dev -tags webkit2_41
```

Migrations run automatically at startup. A pre-migration backup is written to the backup directory before an existing database is upgraded.

If the database file is missing, the app creates a new database and seeds the default `admin` account. Leftover `-wal` or `-shm` files from an interrupted run are moved into `backups/pre-initialize-<timestamp>/` first, so they cannot be attached to the new database.

On Windows the executable folder is the default. If that folder cannot be created or written, the app falls back to the per-user data directory. On first start beside the executable, an existing database and its backups are copied from `%LOCALAPPDATA%\VaccineInventory` when the executable folder has no database yet. On Linux and macOS the per-user data directory is the default, with the executable folder as the fallback. An explicit `VACCINESHIT_DATA_DIR` override never falls back, so tests and development runs stay where they were pointed.

---

## Monthly workflow

1. **Create vaccines.** Add each vaccine's name, optional code, category and unit. Archive rather than delete a vaccine that has history.
2. **Create patients.** A full name is required. A patient code is optional and intended for a local clinic file number, not a national identifier.
3. **Initialize monthly stock.** For each vaccine in the selected month, open the row and explicitly set opening leftover and received stock. Zero is valid. Until this row exists, usage is blocked for that vaccine and month.
   - The suggested opening is the previous month's closing balance (closed month) or derived remaining balance (open month).
   - Changing the suggestion requires an override reason.
4. **Record usage.** Click the numbered day cell and enter the patient, vaccine and number of doses. Usage is whole doses only and cannot exceed available stock.
5. **Correct mistakes.** Edit an open-month entry with a reason, or void it. History is retained and audited.
6. **Adjust received stock.** Use an audited adjustment with a reason when a delivery count was wrong. Adjustments cannot produce negative stock.
7. **Close the month.** Closing stores each vaccine's closing balance. Routine stock and usage edits are then rejected.
8. **Reopen only when necessary.** Reopening requires a reason and is recorded in the audit log. Close the month again after corrections.
9. **Review reports.** Reports show monthly totals, per-vaccine comparisons, daily usage and a forecast. Forecasts use the average of the three most recent complete months, falling back to the most recent complete month, and are planning aids only. Print the monthly report, or open any vaccine row and print its individual report, to A4 paper or PDF; choose «الطباعة إلى ملف» in the print dialog to save a PDF.
10. **Export carefully.** Aggregate CSV is suitable for internal reporting. Patient-level CSV contains confidential health information and must remain local unless an approved process says otherwise.

### Month states

- **Not initialized:** no monthly inventory row for a vaccine; usage is rejected.
- **Open:** stock and usage can change. A public snapshot is provisional and requires explicit confirmation before publishing.
- **Closed:** routine edits are rejected. The stored closing balance is used for the next month's carryover suggestion.

---

## Backups, restore and retention

Backups are created with SQLite `VACUUM INTO`, validated with `PRAGMA integrity_check` and `PRAGMA foreign_key_check`, hashed with SHA-256, and moved into place atomically. The live database file is never copied blindly.

- **Create a backup:** **Settings → Create verified backup…**. Choose a destination outside the live database directory.
- **Restore:** **Settings → Restore backup…**. The chosen file is validated and staged first. The current database is preserved as `backups/pre-restore-<timestamp>.db`. If the restored database cannot be opened, the pre-restore database is put back automatically and the failed copy is retained for diagnosis.
- **Default directory:** `<data dir>/backups`.
- **Audit:** backup creation is recorded in the local audit log.

There is **no automatic retention or deletion**. Recommended operating policy:

- Take a backup before closing a month, before restoring, and before upgrading the application.
- Keep at least daily backups for the last 14 days, plus one backup at each month close, or follow the retention policy of the responsible organization.
- Store copies on encrypted media in a separate location.
- Periodically test a restore on a non-production workstation.
- Treat backups and exports as patient-level confidential data even though the public payload is aggregate-only.

The application log is designed to contain technical errors rather than patient names, usage notes or credentials, but it may still contain local file paths. Review and rotate it under the same retention policy.

---

## Publishing service

Publishing is optional. This build does not include the desktop **Publish** or **Audit** screens. The publication backend and the separate `publicd` service remain available for authorized clients that upload reviewed aggregate snapshots to the API described below.

### 1. Run `publicd`

Build and run the separate service:

```sh
go build -o publicd ./cmd/publicd
PUBLIC_ADMIN_TOKEN="$(openssl rand -hex 32)" \
PUBLIC_DATA_DIR=/var/lib/vaccineshit/public \
PUBLIC_SITE_DIR=/srv/vaccineshit/public \
PUBLIC_LISTEN_ADDR=127.0.0.1:8080 \
./publicd
```

The static public site is the checked-in `public/` directory. Keep its files external (`app.js`, `styles.css`); the service sends a strict Content Security Policy that disallows inline scripts and styles.

#### Environment variables

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `PUBLIC_ADMIN_TOKEN` | Yes | — | Bearer token for snapshot upload and unpublish; at least 24 characters. |
| `PUBLIC_DATA_DIR` | No | `data/public` | Private directory for the current snapshot and published history. |
| `PUBLIC_SITE_DIR` | No | `public` | Static site directory. Missing directory disables static serving with a warning. |
| `PUBLIC_LISTEN_ADDR` | No | `:8080` | Listen address. |
| `PUBLIC_RATE_LIMIT` | No | `120` | Public requests per minute per client IP. |
| `PUBLIC_ADMIN_RATE_LIMIT` | No | `30` | Admin requests per minute per client IP. |
| `PUBLIC_TRUST_PROXY` | No | `false` | Use the first `X-Forwarded-For` address. Enable only behind a trusted proxy that overwrites the header. |
| `PUBLIC_TLS_CERT` | No | — | PEM certificate path. Must be set together with `PUBLIC_TLS_KEY`. |
| `PUBLIC_TLS_KEY` | No | — | PEM private-key path. |

Without `PUBLIC_TLS_CERT` and `PUBLIC_TLS_KEY`, the service serves plain HTTP and logs a warning. Terminate TLS at a trusted reverse proxy in that case. Do not expose plain HTTP to the internet.

#### Endpoints

| Method | Path | Authentication | Purpose |
| --- | --- | --- | --- |
| `PUT` | `/api/admin/snapshot` | Bearer token | Publish an aggregate snapshot. `Content-Type: application/json`. Optional `Idempotency-Key` must equal `snapshotId`. |
| `DELETE` | `/api/admin/snapshot` | Bearer token | Unpublish the current snapshot. |
| `GET` | `/api/public/snapshot` | None | Current published snapshot. Supports `ETag` / `If-None-Match`. |
| `GET` | `/healthz` | None | Liveness check. |
| `GET` | `/` | None | Static public site. |

Admin requests use `Cache-Control: no-store`; public snapshots are cacheable for 60 seconds. The service applies per-IP token buckets, a 256 KiB body limit, strict JSON decoding with unknown-field rejection, and monthly-only snapshot validation.

`GET /api/public/snapshot` returns the flat aggregate snapshot shown above, not the internal publication record. The publication version and timestamp are exposed in the `X-Publication-Version` and `X-Published-At` response headers, and `ETag` supports conditional requests.

#### Publication store and recovery

`PUBLIC_DATA_DIR` contains:

```text
current.json            # atomically replaced pointer to the current snapshot
published/<id>.json     # historical snapshots, retained for idempotency
```

`current.json` is written through a temporary file and atomic rename, so a crash cannot leave a half-written public snapshot. The `published/` history is retained indefinitely so retries stay idempotent; it contains only public aggregate snapshots and there is no automatic pruning. To recover:

- **No current snapshot is intended:** `DELETE /api/admin/snapshot`, or remove `current.json` while the service is stopped.
- **Restore a previous snapshot:** stop the service, copy the desired `published/<id>.json` to `current.json`, then start the service. Keep file ownership and `0600` permissions.
- **Lost the data directory:** republish through the snapshot API. Local publication history records the endpoint, version and payload hash.
- **Rotate the token:** change `PUBLIC_ADMIN_TOKEN` on the service and update the token held by the publishing client. The old token stops working immediately.
- **Back up the public service:** back up `PUBLIC_DATA_DIR` and the static site directory. The public service has no access to the desktop database.

---

## Development

### Repository layout

```text
main.go                     Wails entry point and embedded frontend assets
cmd/publicd/                Separate public HTTP service
internal/app/               Wails binding layer, sanitized DTOs, orchestration
internal/appdata/           OS data-directory resolution
internal/audit/             Audit event types and recording
internal/backup/            Consistent backup and staged restore
internal/domain/            Month/date/quantity validation and balance rules
internal/publicapi/         Public HTTP handler, auth, rate limits, file store
internal/publishing/        Aggregate snapshot boundary, hashing, HTTPS client
internal/reporting/         Pure forecast calculations
internal/service/           Domain services and transaction boundaries
internal/store/sqlite/      SQLite connection, migrations, serialized writes
internal/store/queries/     sqlc source queries
internal/store/generated/   sqlc-generated Go code
migrations/                 SQL schema and generated Go migration map
frontend/src/               React application
frontend/wailsjs/           Generated Wails bindings (do not hand-edit)
public/                     Static public statistics site
```

### Verification

```sh
cd frontend
npm ci
npm run typecheck
npm run lint
npm test
npm run build
cd ..

go generate ./migrations
go vet . ./cmd/... ./internal/... ./migrations/...
go test -race . ./cmd/... ./internal/... ./migrations/...
sqlc diff
```

For a production audit, also run:

```sh
staticcheck ./...
govulncheck ./...
cd frontend && npm audit --omit=dev
```

The root Go package embeds `frontend/dist`, so build the frontend before running `go vet`, `go test` or a plain `go build` on a fresh checkout. `wails dev` and `wails build` perform that step automatically.

Package paths above exclude `frontend/node_modules`, which can contain unrelated Go files and is not part of the application.

### Changing the schema

1. Add a new ordered SQL file under `migrations/` (for example `0002_add_column.sql`). Do not edit an already-applied migration.
2. Run `go generate ./migrations` to regenerate `migrations/migrations_generated.go`.
3. Update `internal/store/queries/*.sql` and run `sqlc generate`.
4. Run `sqlc diff`, `go vet . ./cmd/... ./internal/... ./migrations/...` and `go test . ./cmd/... ./internal/... ./migrations/...`.
5. Start the app against a copy of a real-shaped database and confirm the pre-migration backup and upgrade.

### Changing bound methods

Wails generates `frontend/wailsjs/go/app/*` and `frontend/wailsjs/go/models.ts` from exported methods on `internal/app.App`. After adding or changing an exported binding:

```sh
wails build -tags webkit2_41 -nopackage
```

Commit the regenerated bindings with the Go change. Do not edit generated binding files by hand.

### Why Wails v2.17.1 is required

Wails v2.12.0 embeds `golang.org/x/tools` v0.30.0. That version cannot type-check modules built with Go 1.27, so binding generation fails with errors such as:

```text
internal error: package "fmt" without types was imported from "vaccineshit/internal/appdata"
```

Wails v2.17.1 uses a newer `x/tools` and generates bindings successfully. If you use an older CLI, either upgrade it or generate bindings with a matching older Go toolchain. Do not work around this by permanently building with `-skipbindings`; checked-in bindings can become stale.

---

## Troubleshooting

### `Package webkit2gtk-4.0 was not found`

The machine has WebKitGTK 4.1 only. Add the build tag:

```sh
wails build -tags webkit2_41
```

### Binding generation fails with "without types was imported"

Upgrade the Wails CLI to v2.17.1 or newer:

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.1
```

### The app opens to a recovery screen

The database could not be opened or migrated. The screen shows the data-directory and database paths and offers three actions:

- **Retry startup** after fixing free disk space, file permissions, or a process that still holds the database.
- **Restore a verified backup.** The current database is preserved as `backups/pre-restore-<timestamp>.db`, and a failed restore is rolled back automatically.
- **Initialize a new empty database.** Use this only when the existing file is corrupt and no valid backup exists. The app first moves the current database and its `-wal`/`-shm` files into `backups/pre-initialize-<timestamp>/`, then creates an empty database and seeds `admin` / `admin123`. Change that password immediately after signing in.

  If the old file cannot be moved because Windows still holds it open, the app creates `vaccineshit-recovered-<timestamp>.db` beside it and records the active file in `active-database.txt`, so later starts reuse the new database. The old files stay untouched for repair. If the data directory itself cannot be written, the app retries in the per-user data directory and reports the new location.

Do not delete the database file manually before preserving a copy.

### `database is locked`

The app serializes writes in one process, but another process may be holding the database. Close other copies of the app, external SQLite browsers and backup tools, then retry. Do not run two app instances against the same data directory.

### Publishing returns `preview_stale`

The underlying monthly data changed after the preview. Rebuild the preview, review the figures and publish again.

### Publishing returns an authentication error

The publishing client token and `PUBLIC_ADMIN_TOKEN` must match exactly and be at least 24 characters. Update both after rotation. Verify the endpoint is the HTTPS base URL, not a path to `/api/admin/snapshot`.

### The public site says no statistics are published

Nothing is currently published, or no snapshot has been uploaded since the last unpublish. Check `GET /api/public/snapshot` and the public service publication store.

---

## Limitations

- Single-workstation desktop design; role checks exist but there is no multi-user concurrency or remote access.
- Station-wide operations (backups, restore, publishing, settings, user management, audit review, vaccine deletion) require an admin account.
- Backup files remain unencrypted and unsigned, so protect the destination folder.
- Password recovery for a lost admin password still requires a verified backup or removing the local `users` rows so the default account is seeded again on the next start. Admins can reset other users' passwords from Settings.
- No synchronization or remote database access.
- No application-level database encryption; rely on OS permissions and full-disk encryption.
- No automatic backup retention.
- The forecast is an operational planning estimate, not a medical recommendation.
- The public service is a single-process file-backed service intended for one publishing workstation; it is not a multi-tenant platform.
- This project does not include a license file or a compliance certification. Confirm organizational, privacy and medical-record requirements before production use.
