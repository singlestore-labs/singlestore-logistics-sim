FROM golang:1.15.13-buster as builder

WORKDIR /go/src/simulator

# cache dependencies to accelerate most rebuilds
COPY go.mod go.sum ./
RUN go mod download all

COPY . .

RUN go build -o /simulator bin/simulator/main.go

FROM scratch AS bin
COPY --from=builder /simulator /simulator

FROM debian:buster-slim

CMD ["./simulator", "--config", "config.yaml"]
COPY --from=builder /simulator .
COPY config.yaml /config.yaml