FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY reloquent /usr/local/bin/reloquent
EXPOSE 8230
ENTRYPOINT ["reloquent"]
