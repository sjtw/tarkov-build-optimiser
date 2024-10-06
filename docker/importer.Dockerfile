FROM golang:1.23

WORKDIR /usr/src/app

COPY ../go.mod go.sum ./
RUN go mod download && go mod verify

COPY .. .
RUN ls -la

RUN go build -o ./bin/importer ./cmd/importer/*.go

CMD ["sh", "-c", "./bin/importer"]
