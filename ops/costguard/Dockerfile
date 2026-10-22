# DesignDocument/12_Operation.md 12.12.4 / 14_CICDDesign.md 14.6.4
# Cloud Run は linux/amd64。Apple Silicon でビルドしても同じバイナリになるように固定する。
FROM --platform=linux/amd64 golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

ARG RELEASE=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.release=${RELEASE}" \
    -o /costguard .

FROM --platform=linux/amd64 alpine:3.24

RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /costguard /app/costguard

EXPOSE 8080

ENTRYPOINT ["/app/costguard"]
