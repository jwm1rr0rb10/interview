# --- стадия сборки: полный тулчейн Go ---
FROM golang:1.27 AS build
WORKDIR /src

# сначала только go.mod и go.sum: слой с зависимостями кэшируется,
# пока они не меняются
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
ARG VERSION=dev
# CGO_ENABLED=0 — статический бинарник без libc; -trimpath убирает пути машины сборки
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w -X main.version=${VERSION}" \
      -o /out/todo ./httpapi/cmd/todo

# --- финальная стадия: только бинарник ---
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/todo /todo
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/todo"]
