FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.23
RUN addgroup -S arena && adduser -S arena -G arena
WORKDIR /app
COPY --from=builder /out/server ./server
COPY web ./web
USER arena
EXPOSE 8080
CMD ["./server"]
