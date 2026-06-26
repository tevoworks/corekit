FROM golang:1.25-alpine3.20 AS builder
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 go build -o /api ./cmd/api

FROM alpine:3.20.3
RUN apk add --no-cache ca-certificates tzdata wget && \
    adduser -D -u 1001 appuser
WORKDIR /app
COPY --from=builder /api .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/permissions.yaml .
COPY --from=builder /app/internal/modules/iam/email_template.html ./internal/modules/iam/email_template.html
EXPOSE 8080
USER appuser
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/api/health || exit 1
CMD ["./api"]
