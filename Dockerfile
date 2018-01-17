FROM gliderlabs/alpine:3.6
RUN apk-install ca-certificates
COPY bin/pickabot /bin/pickabot
CMD ["pickabot"]
