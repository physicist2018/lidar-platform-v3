FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /identity ./cmd/identity/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /identity /identity
EXPOSE 8090
CMD ["/identity"]
