# mediasys

Powerful and versatile software for publishing photography online.

![Image](https://raw.githubusercontent.com/camden-git/mediasys/refs/heads/master/.github/assets/albumview.webp)

![Image](https://raw.githubusercontent.com/camden-git/mediasys/refs/heads/master/.github/assets/imageview.webp)

## Running

Everything runs with Docker Compose: Postgres, [Silo](https://github.com/pgsty/silo)
(the community-maintained MinIO fork) for object storage, the Go backend and the
web frontend behind nginx.

```sh
cp .env.example .env   # set passwords and JWT_SECRET (e.g. `openssl rand -hex 32`)
mkdir -p ml-models     # drop the face models in here (see below)
docker compose up -d --build
```

The app is served at `http://localhost:8080`. Create the first admin with
`POST /api/setup/initial-admin` (`{"username": "...", "password": "..."}`).
The storage console is at `http://localhost:9001`.

### Storage

All media lives in one S3 bucket; Postgres holds the metadata.

| prefix            | contents                               |
| ----------------- | -------------------------------------- |
| `originals/`      | uploaded files, `originals/<album folder>/<file>` |
| `thumbnails/`     | WebP thumbnails                        |
| `previews/`       | WebP previews                          |
| `album_banners/`  | album, group and collection banners    |
| `album_archives/` | generated album ZIPs                   |

Uploads stream straight to the bucket and are processed in the background
(metadata, thumbnail and preview in one pass, face detection separately). Work
is tracked in the database, so nothing is lost on restart or when queues fill up.
The backend proxies all media with range, ETag and cache headers, so the bucket
never needs to be public.

### Database migrations

The schema is defined by versioned SQL migrations in `database/migrations/`,
embedded into the binary and applied automatically at startup with
[goose](https://github.com/pressly/goose) (see `database/migrate.go`). Multiple
instances can start concurrently; goose takes a Postgres advisory lock so only one
of them actually runs the migrations while the others wait.

To add a migration, create a new file following goose's naming convention, e.g.:

```sh
go run github.com/pressly/goose/v3/cmd/goose@v3.24.3 -dir database/migrations create add_foo_column sql
```

or copy an existing file and bump the numeric prefix by hand. Write the schema
change under `-- +goose Up` and its reverse under `-- +goose Down`. The GORM model
struct tags in `models/` are kept as documentation of the Go <-> SQL mapping, but no
longer drive the schema — update both the migration and the model tags together
when a model changes.

Databases created before goose was introduced (schema built by GORM AutoMigrate, no
`goose_db_version` table) are upgraded automatically: if the version table is absent
but the app schema already exists, startup records migration 1 as applied and then
runs the rest, where `00002_reconcile_legacy_schema.sql` idempotently fixes up
differences from the old schema (table-wide unique constraints replaced by partial
indexes, missing indexes, extension and collation).

`database/migrate_schema_test.go` guards against drift between the two: when
`TEST_DATABASE_URL` is set to an admin connection on a throwaway Postgres server, it
applies the migrations to one scratch database, runs the old GORM AutoMigrate
directly off the models package on another, and fails if the tables or columns
differ, or if an index declared by the models is missing. Foreign keys and extra
indexes that exist only in SQL are allowed. `migrate_upgrade_test.go` covers the
AutoMigrate-era upgrade path and `migrate_fk_test.go` the foreign key delete behavior.

### Face models

Put these in `ml-models/` (or point `MODELS_DIR` elsewhere):

- `retinaface.onnx`
- `arcface.onnx`
- `deploy.prototxt` and `res10_300x300_ssd_iter_140000_fp16.caffemodel` (fallback detector)

Set `FACE_RECOGNITION_ENABLED=false` to run without them. For GPU builds, pass a
CUDA-enabled OpenCV image via `OPENCV_IMAGE` and uncomment the `deploy` block in
`docker-compose.yml`.

## Development

### Backend tests

`go test ./...` always runs the ordinary Go unit tests. Two extra suites need a
real throwaway Postgres (pgvector) server and a real S3-compatible object store,
and are skipped unless the following env vars are set:

- `TEST_DATABASE_URL` — an admin connection string, e.g.
  `postgres://mediasys:mediasys@localhost:5432/postgres?sslmode=disable`. It must
  be able to `CREATE`/`DROP DATABASE` and `CREATE EXTENSION vector`; each test run
  creates its own scratch database(s) and drops them afterwards.
- `TEST_S3_ENDPOINT`, `TEST_S3_ACCESS_KEY`, `TEST_S3_SECRET_KEY` — an S3-compatible
  endpoint (MinIO/Silo). Optional: `TEST_S3_REGION` (default `us-east-1`),
  `TEST_S3_USE_SSL` (default `false`). Each test run creates its own scratch bucket
  and empties/removes it afterwards.

With those set:

- `database/migrate_schema_test.go` checks the versioned SQL migrations produce
  the same schema as the old GORM AutoMigrate path (drift guard).
- `./e2e/...` spins up the real HTTP router (`app.New`, the same constructor
  `main.go` uses) behind an `httptest.Server` and exercises it end-to-end: auth,
  anonymous access being rejected on mutating/debug routes, the standard error
  response shape, album creation/upload/processing/serving, per-album
  permissions, and a few regression tests (private collections, group/collection
  slug reuse after delete, album listing sort/paging). See the doc comment at the
  top of `e2e/main_test.go` for details.

One-liner to start throwaway containers for local test runs (matches what CI uses):

```sh
docker run -d --name mediasys-test-pg -e POSTGRES_USER=mediasys \
  -e POSTGRES_PASSWORD=mediasys -e POSTGRES_DB=mediasys -p 15432:5432 pgvector/pgvector:pg17
docker run -d --name mediasys-test-minio -e MINIO_ROOT_USER=mediasys \
  -e MINIO_ROOT_PASSWORD=change-me-please -p 19000:9000 pgsty/silo:RELEASE.2026-09-16T00-00-00Z \
  server /data --console-address ":9001"

TEST_DATABASE_URL="postgres://mediasys:mediasys@localhost:15432/postgres?sslmode=disable" \
TEST_S3_ENDPOINT="localhost:19000" TEST_S3_ACCESS_KEY=mediasys TEST_S3_SECRET_KEY=change-me-please \
go test ./...

docker rm -f mediasys-test-pg mediasys-test-minio
```

gocv needs OpenCV 4's headers/libs to build; on macOS with Homebrew:
`PKG_CONFIG_PATH=$(brew --prefix opencv@4)/lib/pkgconfig go test ./...`.

Face detection/recognition itself isn't exercised by the e2e tests (no ONNX
models are available in the test environment); uploads run with
`FACE_RECOGNITION_ENABLED=false`, which still covers metadata/thumbnail/preview
processing.

### Frontend checks

```sh
cd web
pnpm install --frozen-lockfile
./node_modules/.bin/tsc -b       # type-check
./node_modules/.bin/eslint .     # lint
./node_modules/.bin/vite build   # build
```

### CI

`.github/workflows/ci.yml` runs the backend suite (inside the same OpenCV base
image the production Dockerfile builds against, with throwaway pgvector/Postgres
and MinIO) and the frontend checks above on every push/PR.

## License

See [LICENSE](https://github.com/camden-git/mediasys/blob/master/LICENSE) for more information regarding the MIT license.
