FROM golang:1.23

WORKDIR /usr/src/app

COPY ../go.mod go.sum ./
RUN go mod download && go mod verify

COPY .. .
RUN ls -la

RUN go build -o ./bin/evaluator ./cmd/evaluator/*.go

CMD ["sh", "-c", "./bin/evaluator"]
