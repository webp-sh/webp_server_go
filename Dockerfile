FROM golang:1.17.4-alpine as builder

ARG IMG_PATH=/opt/pics
ARG EXHAUST_PATH=/opt/exhaust
RUN apk update && apk add alpine-sdk aom-dev && mkdir /build
COPY go.mod /build
RUN cd /build && go mod download

COPY . /build
RUN cd /build && sed -i "s|.\/pics|${IMG_PATH}|g" config.json  \
    && sed -i "s|\"\"|\"${EXHAUST_PATH}\"|g" config.json  \
    && sed -i 's/127.0.0.1/0.0.0.0/g' config.json  \
    && go build -ldflags="-s -w" -o webp-server .



FROM alpine

COPY --from=builder /build/webp-server  /usr/bin/webp-server
COPY --from=builder /build/config.json /etc/config.json

COPY --from=builder /usr/lib/libaom.a /usr/lib/libaom.a
COPY --from=builder /usr/lib/libaom.so /usr/lib/libaom.so
COPY --from=builder /usr/lib/libaom.so.3 /usr/lib/libaom.so.3
COPY --from=builder /usr/lib/libaom.so.3 /usr/lib/libaom.so.3
COPY --from=builder /usr/lib/libaom.so.3.2.0 /usr/lib/libaom.so.3.2.0


WORKDIR /opt
VOLUME /opt/exhaust
CMD ["/usr/bin/webp-server", "--config", "/etc/config.json"]
