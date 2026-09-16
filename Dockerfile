# Stage 1: build the React frontend
FROM node:22-slim AS web-build
WORKDIR /app/web
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web/ ./
RUN npm run build

# Stage 2: compile the Go binary, embedding the frontend build via go:embed
# (cmd/serve/main.go embeds ../../web/dist, so the layout below must match
# the repo's relative paths).
FROM golang:1.23-alpine AS go-build
WORKDIR /app
COPY go.mod ./
RUN go mod download 2>/dev/null || true
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY --from=web-build /app/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -o /carol-ops ./cmd/serve

# Stage 3: minimal runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 carol-ops
COPY --from=go-build /carol-ops /usr/local/bin/carol-ops
USER carol-ops
WORKDIR /app
EXPOSE 8090
ENTRYPOINT ["/usr/local/bin/carol-ops"]
