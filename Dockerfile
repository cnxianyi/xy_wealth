FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/xy-wealth ./cmd/server

FROM alpine:3.22
RUN addgroup -S app && adduser -S -G app app
COPY --from=builder /out/xy-wealth /usr/local/bin/xy-wealth
USER app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/xy-wealth"]
