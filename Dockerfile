FROM golang:1.24-alpine AS builder

WORKDIR /build

COPY servicoa/go.mod servicoa/go.sum ./servicoa/
WORKDIR /build/servicoa
RUN go mod download

COPY servicoa/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/servicoa .

WORKDIR /build
COPY servicob/go.mod servicob/go.sum ./servicob/
WORKDIR /build/servicob
RUN go mod download

COPY servicob/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/servicob .

FROM alpine:latest

COPY --from=builder /app/servicoa /app/servicoa
COPY --from=builder /app/servicob /app/servicob