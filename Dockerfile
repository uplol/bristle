FROM golang:1.16-alpine AS builder

RUN mkdir -p /usr/src/
WORKDIR /usr/src/
COPY go.mod go.sum /usr/src/
RUN go mod download

COPY . /usr/src/

RUN go build -v -o /bin/bristle cmd/bristle/main.go

FROM alpine
COPY --from=builder /bin/bristle /bin/bristle
ENTRYPOINT ["/bin/bristle"]