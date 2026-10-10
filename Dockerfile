FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /relay ./cmd/relay

FROM alpine:3.20
WORKDIR /app
COPY --from=build /relay /app/relay
COPY relay.yml /app/relay.yml
EXPOSE 8080 9090
ENTRYPOINT ["/app/relay", "-config", "/app/relay.yml"]
