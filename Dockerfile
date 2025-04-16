FROM oven/bun:alpine AS frontend


RUN bun install -g @quasar/cli
WORKDIR /frontend

RUN apk add tree
COPY ./frontend/ /frontend/

RUN pwd && tree -d -L 2

RUN bun install --frozen-lockfile


ENV NODE_ENV=production
RUN bun run build

RUN tree /internal/frontend/dist

FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

COPY --from=frontend /internal/frontend/dist/ /app/internal/frontend/dist

RUN CGO_ENABLED=0 go build -o gs-rproxy main.go

FROM alpine:edge AS proxy

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/gs-rproxy /app/proxy

RUN chmod +x /app/proxy & pwd & ls -la

ENTRYPOINT [ "/app/proxy", "-docker" ]
