FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
# The search index is a build artifact produced by services/crawler and baked
# into the image, so a running container has no runtime dependency on the
# crawler, the network, or a database. Rebuilding data means shipping an image.
COPY data/index.json /seed/index.json
ENV KHODROBIN_INDEX=/data/index.json
ENV KHODROBIN_INDEX_FALLBACK=/seed/index.json
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/api"]
