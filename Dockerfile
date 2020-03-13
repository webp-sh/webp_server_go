FROM golang:alpine as builder
RUN apk update ;\
    apk add alpine-sdk ;\
    git clone https://github.com/webp-sh/webp_server_go /build ;\
    cd /build ;\
    sed -i 's/\/path\/to\/exhaust/\/opt\/exhaust/g' config.json ;\
    sed -i 's/\/path\/to\/pics/\/opt\/pics/g' config.json ;\
    sed -i 's/127.0.0.1/0.0.0.0/g' config.json
WORKDIR /build
RUN go build -o webp-server .
FROM alpine
COPY --from=builder /build/webp-server  /usr/bin/webp-server
COPY --from=builder /build/config.json /etc/config.json
WORKDIR /opt
VOLUME /opt/exhaust
CMD ["/usr/bin/webp-server", "--config", "/etc/config.json"]
