FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w -X github.com/rmewborne/jellygate/cmd/server.version=${VERSION}" \
    -o /jellygate ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /jellygate /jellygate
ENTRYPOINT ["/jellygate"]
