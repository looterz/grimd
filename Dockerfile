FROM scratch
COPY grimd /usr/bin/grimd
EXPOSE 53:53/udp
EXPOSE 53:53/tcp
EXPOSE 8080
ENTRYPOINT ["/usr/bin/grimd"]