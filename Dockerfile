FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /jellygate ./cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=build /jellygate /jellygate
ENTRYPOINT ["/jellygate"]
