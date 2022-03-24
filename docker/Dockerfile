FROM alpine:3.15.0 as certs
RUN apk --update add ca-certificates

FROM golang:1.18.0-alpine3.15 AS builder
RUN apk add git bash gcc musl-dev upx git
WORKDIR /app
COPY . .
RUN git submodule update --init
RUN go mod tidy
RUN go test -v ./...
ENV CGO_ENABLED=0
RUN GOARCH=amd64 go build -ldflags "-w -s" -v ./...
RUN upx -9 -o grimd.minify grimd && mv grimd.minify grimd

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/grimd /usr/bin/grimd
EXPOSE 53:53/udp
EXPOSE 53:53/tcp
EXPOSE 8080
ENTRYPOINT ["/usr/bin/grimd"]