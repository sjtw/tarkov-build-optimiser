FROM golang:1.25

WORKDIR /usr/src/app

COPY ../go.mod go.sum ./
RUN go mod download && go mod verify

COPY .. .
RUN ls -la

RUN go build -o ./bin/api ./cmd/api/*.go

CMD ["sh", "-c", "./bin/api"]
