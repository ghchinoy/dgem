FROM node:22-alpine AS ui-builder
WORKDIR /ui
COPY studio/package.json studio/package-lock.json* studio/tsconfig.json studio/vite.config.ts studio/index.html ./
COPY studio/src ./src
RUN npm install --no-audit --no-fund && npm run build

FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui-builder /ui/dist ./studio/dist
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
