FROM golang:1.24-alpine AS builder

WORKDIR /usr/local/src

COPY ["../go.mod", "../go.sum", "./"]
RUN go mod tidy
RUN go mod download

COPY .. /usr/local/src

RUN go build -o ./bin/server cmd/server/main.go
RUN go build -o ./bin/migrator cmd/migrator/main.go

FROM alpine:latest AS runner

COPY --from=builder /usr/local/src/bin/server /
COPY --from=builder /usr/local/src/bin/migrator /

COPY migrations ./migrations
