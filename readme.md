# mediasys

Powerful and versatile software for publishing photography online.

![Image](https://raw.githubusercontent.com/camden-git/mediasys/refs/heads/master/.github/assets/albumview.webp)

![Image](https://raw.githubusercontent.com/camden-git/mediasys/refs/heads/master/.github/assets/imageview.webp)

## Running

Everything runs with Docker Compose: Postgres, [Silo](https://github.com/pgsty/silo)
(the community-maintained MinIO fork) for object storage, the Go backend and the
web frontend behind nginx.

```sh
cp .env.example .env   # set S3_SECRET_KEY, JWT_SECRET (32+ chars, e.g. `openssl rand -hex 32`) and passwords
mkdir -p ml-models     # drop the face models in here (see below)
docker compose up -d --build
```

The app is served at `http://localhost:8080`. Create the first admin with
`POST /api/setup/initial-admin` (`{"username": "...", "password": "..."}`).
The storage console is at `http://localhost:9001` (bound to localhost only).

### Configuration

Compose reads `.env` (see `.env.example`, which lists every variable with its
default) and passes the relevant ones to the backend. The server validates its
configuration at startup and exits with a clear message on invalid values.

| variable | default | notes |
| -------- | ------- | ----- |
| `JWT_SECRET` | none, required | at least 32 characters |
| `S3_ACCESS_KEY`, `S3_SECRET_KEY` | none, required | also the storage root credentials in compose |
| `S3_ENDPOINT`, `S3_BUCKET`, `S3_REGION`, `S3_USE_SSL` | `localhost:9000`, `mediasys`, `us-east-1`, `false` | |
| `DATABASE_URL` | local Postgres | compose builds it from `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` |
| `DATABASE_DEBUG` | `false` | log every SQL statement |
| `PORT` | `8080` | backend listen port (compose publishes nginx on `HTTP_PORT`) |
| `CORS_ALLOWED_ORIGINS` | localhost dev origins | comma-separated |
| `MAX_UPLOAD_SIZE_MB` | `200` | |
| `THUMBNAIL_MAX_SIZE`, `THUMBNAIL_QUEUE_SIZE`, `NUM_THUMBNAIL_WORKERS` | `300`, `50`, `2` | |
| `NUM_DETECTION_WORKERS`, `DETECTION_QUEUE_SIZE` | `1`, `10` | |
| `MEMORY_TRIM_INTERVAL_MINUTES` | `30` | `0` disables the trimmer |
| `GOMEMLIMIT` | unset | Go soft memory limit, e.g. `2GiB` |
| `FACE_RECOGNITION_ENABLED` | `true` | |
| `FACE_RECOGNITION_MODEL_NAME` | `arcface` | |
| `FACE_RECOGNITION_THRESHOLD` | `0.6` | between 0 and 1 |
| `CUDA_ENABLED`, `RETINAFACE_CUDA`, `ARCFACE_CUDA` | `false` in compose | needs a CUDA-enabled OpenCV image |
| `FACE_DNN_CONFIG_PATH`, `FACE_DNN_MODEL_PATH`, `RETINAFACE_MODEL_PATH`, `FACE_RECOGNITION_MODEL_PATH` | `/models/...` in the image | model file locations |
| `TURNSTILE_SITE_KEY`, `TURNSTILE_SECRET_KEY` | empty | Cloudflare Turnstile |
| `OPENCV_IMAGE` | `ghcr.io/hybridgroup/opencv:4.11.0` | build arg for the backend image |
| `STORAGE_CONSOLE_PORT` | `9001` | host port of the storage console |

Invalid values (for example a non-numeric worker count) abort startup instead of
silently falling back to a default.

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
