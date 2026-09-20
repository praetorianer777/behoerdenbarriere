# The Go version follows go.mod; a pin that lags behind it fails the build with
# "go.mod requires go >= …", which is exactly what happened here.
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -trimpath -o /out/worker ./cmd/worker && \
    CGO_ENABLED=0 go build -trimpath -o /out/seed ./cmd/seed && \
    CGO_ENABLED=0 go build -trimpath -o /out/import ./cmd/import && \
    CGO_ENABLED=0 go build -trimpath -o /out/dns ./cmd/dns

FROM gcr.io/distroless/static-debian12 AS api
# The operational commands travel with the API image. An installation that has nothing
# but containers has no Go toolchain, so anything the instructions tell someone to run
# has to be in here — /import was not, and a fresh installation quietly ended up with
# the hand-kept authorities and none of the districts.
#   docker compose run --rm --entrypoint /seed api
#   docker compose run --rm --entrypoint /import api
#   docker compose run --rm --entrypoint /dns api
COPY --from=build /out/api /api
COPY --from=build /out/seed /seed
COPY --from=build /out/import /import
COPY --from=build /out/dns /dns
USER nonroot:nonroot
ENTRYPOINT ["/api"]

FROM gcr.io/distroless/static-debian12 AS worker
COPY --from=build /out/worker /worker
USER nonroot:nonroot
ENTRYPOINT ["/worker"]
