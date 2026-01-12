# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

# Copy pre-generated templ files and pre-built static assets from CI
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/dejaview ./cmd/dejaview

FROM alpine:3.20
ARG LOG_LEVEL=info
WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /out/dejaview ./dejaview
COPY --from=builder /src/static ./static
COPY migrations ./migrations

RUN addgroup -S dejaview \
    && adduser -S -G dejaview dejaview \
    && chown -R dejaview:dejaview /app

ENV PORT=4600
ENV LOG_LEVEL=${LOG_LEVEL}
USER dejaview

EXPOSE 4600
CMD ["./dejaview"]


