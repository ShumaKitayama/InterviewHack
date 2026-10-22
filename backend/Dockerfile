# DesignDocument/14_CICDDesign.md 14.5.2 / 14.6
# Cloud Run は linux/amd64。Apple Silicon でビルドしても同じバイナリになるように固定する。
FROM --platform=linux/amd64 golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG RELEASE=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.release=${RELEASE}" \
    -o /server ./cmd/api

FROM --platform=linux/amd64 alpine:3.24

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /server /app/server

EXPOSE 8080

ENTRYPOINT ["/app/server"]
