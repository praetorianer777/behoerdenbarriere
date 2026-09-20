# The Go version follows go.mod; a pin that lags behind it fails the build with
# "go.mod requires go >= …", which is exactly what happened here.
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -trimpath -o /out/worker ./cmd/worker && \
    CGO_ENABLED=0 go build -trimpath -o /out/seed ./cmd/seed

FROM gcr.io/distroless/static-debian12 AS api
# The seeder travels with the API image so an installation that has nothing but
# containers can still load the list of authorities:
#   docker compose run --rm --entrypoint /seed api
COPY --from=build /out/api /api
COPY --from=build /out/seed /seed
USER nonroot:nonroot
ENTRYPOINT ["/api"]

FROM gcr.io/distroless/static-debian12 AS worker
COPY --from=build /out/worker /worker
USER nonroot:nonroot
ENTRYPOINT ["/worker"]
