FROM golang:alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /build/UnoGoBot .

FROM alpine:latest

RUN apk add --no-cache ca-certificates

COPY --from=builder /build/UnoGoBot /UnoGoBot

CMD ["/UnoGoBot"]
