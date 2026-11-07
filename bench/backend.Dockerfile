FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /backend ./examples/backends

FROM alpine:3.20
COPY --from=build /backend /backend
ENTRYPOINT ["/backend"]
