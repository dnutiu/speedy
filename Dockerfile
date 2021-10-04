FROM golang:1.16 as builder

WORKDIR /app
COPY . .

RUN go mod download
RUN go build -o speedy .

FROM ubuntu

COPY --from=builder /app /app

ENTRYPOINT [ "/app/speedy" ]