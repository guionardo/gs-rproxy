# FROM oven/bun:1 AS frontend

# WORKDIR /app

# COPY frontend/* ./

# RUN bun install --frozen-lockfile

# ENV NODE_ENV=production
# RUN bun run build

# WORKDIR /usr/src/app

# # install dependencies into temp directory
# # this will cache them and speed up future builds
# FROM base AS install
# RUN mkdir -p /temp/dev
# COPY package.json bun.lock /temp/dev/
# RUN cd /temp/dev && bun install --frozen-lockfile

# # install with --production (exclude devDependencies)
# RUN mkdir -p /temp/prod
# COPY package.json bun.lock /temp/prod/
# RUN cd /temp/prod && bun install --frozen-lockfile --production

# # copy node_modules from temp directory
# # then copy all (non-ignored) project files into the image
# FROM base AS prerelease
# COPY --from=install /temp/dev/node_modules node_modules
# COPY . .

# # [optional] tests & build
# ENV NODE_ENV=production
# RUN bun test
# RUN bun run build

# # copy production dependencies and source code into final image
# FROM base AS release
# COPY --from=install /temp/prod/node_modules node_modules
# COPY --from=prerelease /usr/src/app/index.ts .
# COPY --from=prerelease /usr/src/app/package.json .

# # run the app
# USER bun
# EXPOSE 3000/tcp
# ENTRYPOINT [ "bun", "run", "index.ts" ]

FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

# COPY --from=frontend /dist/ /app/internal/frontend/dist

RUN CGO_ENABLED=0 go build -o gs-rproxy main.go

FROM alpine:edge AS proxy

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/gs-rproxy /app/proxy

RUN chmod +x /app/proxy & pwd & ls -la

ENTRYPOINT [ "/app/proxy", "-docker" ]

