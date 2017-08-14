FROM alpine:3.6

RUN apk add --no-cache ca-certificates
COPY pmmap.docker /usr/bin
RUN chmod u+x /usr/bin/pmmap.docker

EXPOSE 8080

ENTRYPOINT ["/usr/bin/pmmap.docker"]
