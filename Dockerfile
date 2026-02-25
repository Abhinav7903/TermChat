# ---------- Build Stage ----------
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o termchat ./cmd

# ---------- Runtime Stage ----------
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/termchat .

EXPOSE 8080

CMD ["./termchat", "-mode=server"]