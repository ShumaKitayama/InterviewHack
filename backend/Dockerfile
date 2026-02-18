# DesignDocument/14_CICDDesign.md 14.5.2 / 14.6
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG RELEASE=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.release=${RELEASE}" \
    -o /server ./cmd/api

FROM alpine:3.21

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /server /app/server

EXPOSE 8080

ENTRYPOINT ["/app/server"]
