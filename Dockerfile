## We specify the base image we need for our
## go application
FROM golang:1.23-alpine AS builder
## We create an /app directory within our
## image that will hold our application source
## files
RUN mkdir /app
## We copy everything in the root directory
## into our /app directory
ADD . /app
## We specify that we now wish to execute
## any further commands inside our /app
## directory
WORKDIR /app
## we run go build to compile the binary
## executable of our Go program
RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o goboilerplate ./cmd

FROM alpine:3.14  

COPY --from=builder /app/goboilerplate .
COPY --from=builder /app/internal/migration/postgres internal/migration/postgres
COPY --from=builder /app/.env.example .env

ENTRYPOINT ["./goboilerplate"]
