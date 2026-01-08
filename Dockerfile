# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

# Copy pre-generated templ files and pre-built static assets from CI
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/seenema ./cmd/seenema

FROM alpine:3.20
ARG LOG_LEVEL=info
WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /out/seenema ./seenema
COPY --from=builder /src/static ./static
COPY migrations ./migrations

RUN addgroup -S seenema \
    && adduser -S -G seenema seenema \
    && chown -R seenema:seenema /app

ENV PORT=4600
ENV LOG_LEVEL=${LOG_LEVEL}
USER seenema

EXPOSE 4600
CMD ["./seenema"]


