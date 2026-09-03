FROM node:24-alpine AS web-builder

WORKDIR /src/web
RUN npm install --global pnpm@10.33.4
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

FROM golang:1.25-alpine AS server-builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/xy-wealth ./cmd/server

FROM alpine:3.22
RUN addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=server-builder /out/xy-wealth /usr/local/bin/xy-wealth
COPY --from=web-builder /src/web/dist /app/web
ENV XY_WEALTH_WEB_STATIC_DIR=/app/web
USER app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/xy-wealth"]
