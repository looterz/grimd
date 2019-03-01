FROM golang:1.12.0-alpine3.9
LABEL mantainer="integrii@gmail.com"

ADD *.go /go/src/github.com/looterz/grimd/
WORKDIR /go/src/github.com/looterz/grimd
RUN apk --no-cache add git upx
RUN go get -d
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -a -installsuffix cgo .
RUN upx -9 grimd
RUN cp grimd /grimd

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /grimd .

EXPOSE 53:53/udp
EXPOSE 53:53/tcp
EXPOSE 8080

ENTRYPOINT ["./grimd"]
