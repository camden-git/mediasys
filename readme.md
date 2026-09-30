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
mkdir -p ml-models     # drop the face models in here
docker compose up -d --build
```

The app is served at `http://localhost:8080`. Create the first admin with:

```sh
docker compose exec backend /app/mediasys user create --admin
```

The storage console is at `http://localhost:9001` (bound to localhost only).

## License

See [LICENSE](https://github.com/camden-git/mediasys/blob/master/LICENSE) for more information regarding the MIT license.
