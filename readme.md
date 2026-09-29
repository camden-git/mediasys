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

`database/migrate_schema_test.go` guards against drift between the two: when
`TEST_DATABASE_URL` is set to an admin connection on a throwaway Postgres server, it
applies the migrations to one scratch database, runs the old GORM AutoMigrate
directly off the models package on another, and fails if the resulting tables,
columns or indexes differ.

### Face models

Put these in `ml-models/` (or point `MODELS_DIR` elsewhere):

- `retinaface.onnx`
- `arcface.onnx`
- `deploy.prototxt` and `res10_300x300_ssd_iter_140000_fp16.caffemodel` (fallback detector)

Set `FACE_RECOGNITION_ENABLED=false` to run without them. For GPU builds, pass a
CUDA-enabled OpenCV image via `OPENCV_IMAGE` and uncomment the `deploy` block in
`docker-compose.yml`.

## License

See [LICENSE](https://github.com/camden-git/mediasys/blob/master/LICENSE) for more information regarding the MIT license.
