# syntax=docker/dockerfile:1

# OpenCV base used to build and to source runtime libraries from.
# Override for GPU builds, e.g. --build-arg OPENCV_IMAGE=<cuda-enabled opencv image>.
ARG OPENCV_IMAGE=ghcr.io/hybridgroup/opencv:4.11.0
ARG GO_IMAGE=golang:1.25-bookworm
ARG RUNTIME_IMAGE=debian:bullseye-slim

FROM ${GO_IMAGE} AS go

# --- build -------------------------------------------------------------------
FROM ${OPENCV_IMAGE} AS builder
COPY --from=go /usr/local/go /opt/go
# prefer the OpenCV image's own libs (e.g. its oneTBB) over older distro copies
ENV PATH=/opt/go/bin:$PATH \
    GOTOOLCHAIN=local \
    CGO_ENABLED=1 \
    LD_LIBRARY_PATH=/usr/local/lib \
    CGO_LDFLAGS="-L/usr/local/lib -Wl,-rpath-link,/usr/local/lib"

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags "-s -w" -o /out/mediasys .

# collect every shared library the binary (and wget, for the healthcheck) links against
RUN mkdir -p /deps/usr/bin && cp /usr/bin/wget /deps/usr/bin/ && \
    for bin in /out/mediasys /usr/bin/wget; do \
      ldd "$bin" | awk '/=> \// {print $3}' | xargs -I{} cp --parents -L {} /deps; \
    done

# --- runtime -----------------------------------------------------------------
FROM ${RUNTIME_IMAGE}

# bullseye's apt mirrors are archived, so copy what we need from the builder instead
COPY --from=builder /etc/ssl/certs /etc/ssl/certs
COPY --from=builder /deps/ /
COPY --from=builder /out/mediasys /app/mediasys
RUN ldconfig && useradd -m -u 10001 appuser

ENV PORT=8080 \
    FACE_DNN_CONFIG_PATH=/models/deploy.prototxt \
    FACE_DNN_MODEL_PATH=/models/res10_300x300_ssd_iter_140000_fp16.caffemodel \
    RETINAFACE_MODEL_PATH=/models/retinaface.onnx \
    FACE_RECOGNITION_MODEL_PATH=/models/arcface.onnx

WORKDIR /app
USER appuser
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/api/healthz || exit 1

CMD ["/app/mediasys"]
