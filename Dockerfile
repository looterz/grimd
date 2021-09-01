FROM golang:alpine
LABEL mantainer="integrii@gmail.com"

ADD *.* /go/src/github.com/looterz/grimd/
WORKDIR /go/src/github.com/looterz/grimd
RUN \
  apk --no-cache add git && \
  go get -v && \
  go build -v && \
  mkdir /app && \
  cp grimd /app/grimd
WORKDIR /app

EXPOSE 53:53/udp
EXPOSE 53:53/tcp
EXPOSE 8080

ENTRYPOINT ["/app/grimd"]
