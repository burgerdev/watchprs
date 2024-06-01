FROM golang:1.22.3 AS builder
ADD . /src
WORKDIR /src
RUN CGO_ENABLED=0 go build -a -o watchprs main.go

FROM alpine:latest
COPY --from=builder /src/watchprs /watchprs
ENTRYPOINT ["/watchprs"]
