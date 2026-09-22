FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /out/dgem .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/dgem /app/dgem
COPY templates /app/templates
ENV PORT=8080
ENV DGEM_GCP_AUTH=1
EXPOSE 8080
ENTRYPOINT ["/app/dgem", "serve", "--templates-dir", "/app/templates"]
