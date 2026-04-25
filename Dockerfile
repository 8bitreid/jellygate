FROM golang:1.25-alpine AS build
WORKDIR /src

# Install Node.js for building CSS
RUN apk add --no-cache nodejs npm

# Copy package files and install dependencies
COPY package.json package-lock.json* ./
RUN npm ci --only=production --ignore-scripts || npm install --only=production --ignore-scripts

# Copy Tailwind config and input CSS
COPY tailwind.config.js ./
COPY web/static/input.css ./web/static/

# Build CSS
RUN npm run build:css

# Copy Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy remaining source files
COPY . .

# Build the Go binary
ARG VERSION=dev
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w -X github.com/rmewborne/jellygate/cmd/server.version=${VERSION}" \
    -o /jellygate ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /jellygate /jellygate
ENTRYPOINT ["/jellygate"]
