# syntax=docker/dockerfile:1

# --- Builder stage -----------------------------------------------------------
# Uses an image with OpenCV libs and headers preinstalled for GoCV builds.
# See: https://github.com/hybridgroup/gocv/blob/release/Dockerfile
FROM ghcr.io/hybridgroup/gocv:0.41.0-cuda11.8-ubuntu22.04 AS builder

WORKDIR /src

# Install build prerequisites for Go modules (git, ca-certs)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates git && \
    rm -rf /var/lib/apt/lists/*

# Copy go.mod/go.sum first for cached deps
COPY go.mod go.sum ./
RUN go mod download

# Copy the whole source
COPY . .

# Build static-ish binary (still dynamically links to OpenCV in runtime image)
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o /app/mediasysbackend ./

# --- Runtime stage -----------------------------------------------------------
# Use the matching runtime image that includes OpenCV .so libs for GoCV.
FROM ghcr.io/hybridgroup/gocv:0.41.0-cuda11.8-ubuntu22.04

WORKDIR /app

# Create non-root user
RUN useradd -m -u 10001 appuser

# Copy binary
COPY --from=builder /app/mediasysbackend /app/mediasysbackend

# Create directories typically used by the app; bind mount at runtime as needed
RUN mkdir -p /data/media_storage /data/models /data/db

# Default environment (override via docker-compose or -e)
ENV ROOT_DIRECTORY=/data \
    MEDIA_STORAGE_PATH=/data/media_storage \
    DATABASE_PATH=/data/db/images.db \
    THUMBNAILS_SUBDIR=thumbnails \
    BANNERS_SUBDIR=album_banners \
    ARCHIVES_SUBDIR=album_archives \
    FACE_DNN_CONFIG_PATH=/data/models/deploy.prototxt \
    FACE_DNN_MODEL_PATH=/data/models/res10_300x300_ssd_iter_140000_fp16.caffemodel \
    RETINAFACE_MODEL_PATH=/data/models/retinaface.onnx \
    FACE_RECOGNITION_MODEL_PATH=/data/models/arcface.onnx \
    FACE_RECOGNITION_MODEL_NAME=arcface \
    FACE_RECOGNITION_ENABLED=true \
    PORT=8080

# Expose API port
EXPOSE 8080

# Use non-root
USER appuser

# Healthcheck
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD wget -qO- http://127.0.0.1:8080/api/permissions/keys || exit 1

# Run
CMD ["/app/mediasysbackend"]



