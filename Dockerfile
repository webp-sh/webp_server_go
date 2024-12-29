FROM golang:1.23-bookworm AS builder

ARG IMG_PATH=/opt/pics
ARG EXHAUST_PATH=/opt/exhaust
RUN apt update && apt install --no-install-recommends libvips-dev -y && mkdir /build
COPY go.mod /build
RUN cd /build && go mod download

COPY . /build
RUN cd /build && sed -i "s|.\/pics|${IMG_PATH}|g" config.json  \
    && sed -i "s|\"\"|\"${EXHAUST_PATH}\"|g" config.json  \
    && sed -i 's/127.0.0.1/0.0.0.0/g' config.json  \
    && go build -ldflags="-s -w" -o webp-server .

FROM debian:bookworm-slim

RUN apt update && apt install --no-install-recommends libvips ca-certificates libjemalloc2 libtcmalloc-minimal4 curl -y && rm -rf /var/lib/apt/lists/* &&  rm -rf /var/cache/apt/archives/*

COPY ./assets /build/assets
# Install libam with correct arch
RUN dpkg -i /build/assets/libaom3_3.11.0-1_$(dpkg --print-architecture).deb && \
    rm /build/assets/libaom3_3.11.0-1_$(dpkg --print-architecture).deb

COPY --from=builder /build/webp-server  /usr/bin/webp-server
COPY --from=builder /build/config.json /etc/config.json

WORKDIR /opt
VOLUME /opt/exhaust
CMD ["/usr/bin/webp-server", "--config", "/etc/config.json"]
