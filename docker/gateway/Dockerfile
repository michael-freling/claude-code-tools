# Build stage
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /claude-forge ./cmd/claude-forge/

# Runtime stage
FROM alpine:3.21
RUN apk add --no-cache bash openssh-client git
COPY --from=build /claude-forge /usr/local/bin/claude-forge
RUN adduser -D -s /bin/bash user
USER user
ENTRYPOINT ["claude-forge", "gateway"]
