FROM alpine:3.10
RUN apk add ca-certificates && update-ca-certificates
COPY bin/pickabot /bin/pickabot
CMD ["pickabot"]
