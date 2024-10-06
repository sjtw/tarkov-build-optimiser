FROM golang:1.23

WORKDIR /usr/src/app

COPY ../go.mod go.sum ./
RUN go mod download && go mod verify

COPY .. .
RUN ls -la

RUN go build -o ./bin/migrate ./cmd/migrations/*.go

CMD ["sh", "-c", "./bin/migrate postgres \"host=$POSTGRES_HOST user=$POSTGRES_USER password=$POSTGRES_PASSWORD dbname=$POSTGRES_DB port=5432 sslmode=disable\" up"]
