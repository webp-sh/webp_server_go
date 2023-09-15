FROM golang:1.21-bookworm as builder

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

FROM debian:bookworm-slim as libwebp
RUN apt update && apt install -y wget gcc make autoconf automake libtool libgif-dev \
    libjpeg-dev libjpeg62-turbo libjpeg62-turbo-dev libpng-dev libpng-tools libpng16-16 libtiff-dev libtiff6 libtiffxx6
RUN mkdir libwebp && mkdir -p /build/usr && mkdir /build/usr/lib/ && cd libwebp &&  \
        wget https://chromium.googlesource.com/webm/libwebp/+archive/refs/heads/1.3.2.tar.gz && \
        tar xf 1.3.2.tar.gz && rm -f 1.3.2.tar.gz && \
        ./autogen.sh && \
        ./configure --prefix=/build/usr --libdir=/build/usr/lib --enable-everything && \
        make && make install

FROM debian:bookworm-slim

RUN apt update && apt install --no-install-recommends libvips ca-certificates libjemalloc2 libtcmalloc-minimal4 -y && rm -rf /var/lib/apt/lists/* &&  rm -rf /var/cache/apt/archives/*

# for CVE-2023-4863
RUN dpkg --remove --force-depends libwebp-dev libwebp7 libwebpdemux2 libwebpmux3
COPY --from=libwebp /build/usr/lib/* /usr/lib/temp-linux-gnu/
RUN mv /usr/lib/temp-linux-gnu/* /usr/lib/$(uname -m)-linux-gnu/ && ldconfig

COPY --from=builder /build/webp-server  /usr/bin/webp-server
COPY --from=builder /build/config.json /etc/config.json

WORKDIR /opt
VOLUME /opt/exhaust
CMD ["/usr/bin/webp-server", "--config", "/etc/config.json"]
