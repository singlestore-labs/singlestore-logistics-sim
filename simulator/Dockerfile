FROM golang:1.15

WORKDIR /go/src/simulator
COPY . .

RUN go mod download
RUN go get github.com/githubnemo/CompileDaemon

RUN go install -v ./...

CMD CompileDaemon --build="go install ./..." --command="simulator"