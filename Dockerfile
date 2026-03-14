FROM node:20-alpine AS web-builder

WORKDIR /src/web

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/index.html ./index.html
COPY web/vite.config.js ./vite.config.js
COPY web/src ./src

ENV VITE_API_BASE_URL=/

RUN npm run build

FROM golang:1.22-alpine AS builder

WORKDIR /src/server

RUN apk add --no-cache ca-certificates tzdata

COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/cmd ./cmd
COPY server/internal ./internal
COPY server/migrations ./migrations

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/proman ./cmd/api

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S -G app app

WORKDIR /app

COPY --from=builder /out/proman ./proman
COPY --from=builder /src/server/migrations ./migrations
COPY --from=web-builder /src/web/dist ./web/dist

ENV APP_ENV=production
ENV HTTP_PORT=8080

EXPOSE 8080

USER app

CMD ["/app/proman"]
