FROM golang:1.15

WORKDIR /go/src/simulator
COPY . .

RUN go mod download && \
    go get github.com/githubnemo/CompileDaemon && \
    go install -v ./...

CMD ["CompileDaemon", "--build='go install ./...'", "--command='simulator --config config.yaml'"]
