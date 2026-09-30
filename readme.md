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
