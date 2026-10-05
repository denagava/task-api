FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /task-api .

FROM alpine:3.20
RUN adduser -D -u 1000 app
USER app
COPY --from=builder /task-api /usr/local/bin/task-api
EXPOSE 8080
ENTRYPOINT ["task-api"]
