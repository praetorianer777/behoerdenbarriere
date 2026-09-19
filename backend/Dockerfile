FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -trimpath -o /out/worker ./cmd/worker

FROM gcr.io/distroless/static-debian12 AS api
COPY --from=build /out/api /api
USER nonroot:nonroot
ENTRYPOINT ["/api"]

FROM gcr.io/distroless/static-debian12 AS worker
COPY --from=build /out/worker /worker
USER nonroot:nonroot
ENTRYPOINT ["/worker"]
