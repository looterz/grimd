FROM alpine:3.15.0 as certs
RUN apk --update add ca-certificates

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY grimd /usr/bin/grimd
EXPOSE 53/udp
EXPOSE 53/tcp
EXPOSE 8080/tcp
ENTRYPOINT ["/usr/bin/grimd"]