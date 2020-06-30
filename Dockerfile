FROM golang:alpine as builder
ARG IMG_PATH=/opt/pics
ARG EXHAUST_PATH=/opt/exhaust
RUN apk update ;\
    apk add alpine-sdk ;\
    git clone https://github.com/webp-sh/webp_server_go /build ;\
    cd /build ;\
    sed -i "s|.\/pics|${IMG_PATH}|g" config.json ;\
    sed -i "s|\"\"|\"${EXHAUST_PATH}\"|g" config.json ;\
    sed -i 's/127.0.0.1/0.0.0.0/g' config.json
WORKDIR /build
RUN go build -o webp-server .
FROM alpine
COPY --from=builder /build/webp-server  /usr/bin/webp-server
COPY --from=builder /build/config.json /etc/config.json
WORKDIR /opt
VOLUME /opt/exhaust
CMD ["/usr/bin/webp-server", "--config", "/etc/config.json"]