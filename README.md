<p align="center">
	<img src="./pics/webp_server.png"/>
</p>

[![CI](https://github.com/webp-sh/webp_server_go/actions/workflows/CI.yaml/badge.svg)](https://github.com/webp-sh/webp_server_go/actions/workflows/CI.yaml)
[![build docker image](https://github.com/webp-sh/webp_server_go/actions/workflows/release_binary.yaml/badge.svg)](https://github.com/webp-sh/webp_server_go/actions/workflows/release_binary.yaml)
[![Release WebP Server Go Binaries](https://github.com/webp-sh/webp_server_go/actions/workflows/release_docker_image.yaml/badge.svg)](https://github.com/webp-sh/webp_server_go/actions/workflows/release_docker_image.yaml)
[![codecov](https://codecov.io/gh/webp-sh/webp_server_go/branch/master/graph/badge.svg?token=VR3BMZME65)](https://codecov.io/gh/webp-sh/webp_server_go)
![Docker Pulls](https://img.shields.io/docker/pulls/webpsh/webp-server-go?style=plastic)

[Documentation](https://docs.webp.sh/) | [Website](https://webp.sh/) | [Blog](https://blog.webp.se/)

This is a Server based on Golang, which allows you to serve WebP images on the fly.

Currently supported image format: JPEG, PNG, BMP, GIF, SVG, HEIC, NEF, WEBP

> e.g When you visit `https://your.website/pics/tsuki.jpg`，it will serve as `image/webp`/`image/avif` format without changing the URL.
>
> GIF image will not be converted to AVIF format because the converted AVIF image is not animated.

## Usage with Docker(recommended)

We strongly recommend using Docker to run WebP Server Go because running it directly with the binary may encounter issues with `glibc` and some dependency libraries, which can be quite tricky to resolve.

Make sure you've got Docker and `docker-compose` installed, create a directory and create `docker-compose.yml` file inside it like this:

```yml
version: '3'

services:
  webp:
    image: webpsh/webp-server-go
    # image: ghcr.io/webp-sh/webp_server_go
    restart: always
    volumes:
      - ./path/to/pics:/opt/pics
      - ./exhaust:/opt/exhaust
      - ./metadata:/opt/metadata
    ports:
      -  127.0.0.1:3333:3333
```

Suppose your website and image has the following pattern.

| Image Path                            | Website Path                         |
| ------------------------------------- | ------------------------------------ |
| `/var/www/img.webp.sh/path/tsuki.jpg` | `https://img.webp.sh/path/tsuki.jpg` |

Then

* `./path/to/pics` should be changed to `/var/www/img.webp.sh`
* `./exhaust` is cache folder for output images, by default it will be in `exhaust` directory alongside with `docker-compose.yml` file, if you'd like to keep cached images in another folder, you can change  `./exhaust` to `/some/other/path/to/exhaust`
* `./metadata` is cache folder for images' metadata, by default it will be in `metadata` directory alongside with `docker-compose.yml` file

Start the container using:

```
docker-compose up -d
```

Now the server should be running on `127.0.0.1:3333`, visiting `http://127.0.0.1:3333/path/tsuki.jpg` will see the optimized version of `/var/www/img.webp.sh/path/tsuki.jpg`, you can now add reverse proxy to make it public, for example, let Nginx to `proxy_pass http://127.0.0.1:3333/;`, and your WebP Server is on-the-fly!

## Custom config

If you'd like to use a customized `config.json`, you can follow the steps in [Configuration | WebP Server Documentation](https://docs.webp.sh/usage/configuration/) to genereate one, and mount it into the container's `/etc/config.json`, example `docker-compose.yml` as follows:

```yml
version: '3'

services:
  webp:
    image: webpsh/webp-server-go
    # image: ghcr.io/webp-sh/webp_server_go
    restart: always
    volumes:
      - ./path/to/pics:/opt/pics
      - ./path/to/exhaust:/opt/exhaust
      - ./path/to/metadata:/opt/metadata
      - ./config.json:/etc/config.json
    ports:
      -  127.0.0.1:3333:3333
```

You can refer to [Configuration | WebP Server Documentation](https://docs.webp.sh/usage/configuration/) for more info, such as custom config, AVIF support etc.

## Advanced Usage

If you'd like to use with binary, please consult to [Use with Binary(Advanced) | WebP Server Documentation](https://docs.webp.sh/usage/usage-with-binary/)

> spoiler alert: you may encounter issues with `glibc` and some dependency libraries.

For `supervisor` or detailed Nginx configuration, please read our documentation at [https://docs.webp.sh/](https://docs.webp.sh/)

## WebP Cloud Services

We are currently building a new service called [WebP Cloud Services](https://webp.se/), it now has three parts:

* [Public Service](https://public.webp.se)
  * GitHub Avatar/Gravater reverse proxy with WebP optimization, for example, change `https://www.gravatar.com/avatar/09eba3a443a7ea91cf818f6b27607d66` to `https://gravatar.webp.se/avatar/09eba3a443a7ea91cf818f6b27607d66` for rendering will get a smaller version of gravater, making your website faster 
  * Totally free service and currently has a large number of users, this includes, but is not limited to [CNX Software](https://www.cnx-software.com/), [Indienova](https://indienova.com/en) 
* [WebP Cloud](https://docs.webp.se/webp-cloud/)
  * No need to install WebP Server Go yourself, especially suitable for static websites.
  * Image Conversion: WebP Cloud converts images to WebP/AVIF format, reducing size while maintaining quality for faster website loading.
    * Example 1: Original image URL (https://yyets.dmesg.app/api/user/avatar/BennyThink) becomes compressed URL (https://vz4w427.webp.ee/api/user/avatar/Benny).
    * Example 2: Original image URL (https://yyets.dmesg.app/api/user/avatar/BennyThink) becomes a thumbnail image using URL (https://vz4w427.webp.ee/api/user/avatar/BennyThink?width=200).
  * Caching: WebP Cloud automatically caches served images, reducing traffic and bandwidth load on the origin server.
* [Fly](https://webp.se/fly/)
  * We call this service Fly, with the aim of providing a public and free service that users can experience without registering on WebP Cloud.
    As this is a public service, some limitations compared to WebP Cloud are imposed:

    - Fly supports a maximum original image size of 8MB, while WebP Cloud supports up to 80MB.
    - Fly cache time is 1 day, while WebP Cloud has unlimited time (can be manually cleared at any time).
    - It does not support parameters like `blur`, `sharpen` for image processing.
    - And that’s it.

For detailed information, please visit [WebP Cloud Services Website](https://webp.se/) or [WebP Cloud Services Docs](https://docs.webp.se/).

## Support us

If you find this project useful, please consider supporting
us by [becoming a sponsor](https://github.com/sponsors/webp-sh), pay via Stripe, or [try out our WebP Cloud](https://docs.webp.se/webp-cloud/basic/)!

| USD(Card, Apple Pay and Google Pay)              | EUR(Card, Apple Pay and Google Pay)              | CNY(Card, Apple Pay, Google Pay and Alipay)      |
|--------------------------------------------------|--------------------------------------------------|--------------------------------------------------|
| [USD](https://donate.stripe.com/4gwfZn2RDgag0bScMN) | [EUR](https://donate.stripe.com/28odRfgItgage2IfZ0) | [CNY](https://donate.stripe.com/00geVj8bX3nuf6MeUU) |
| ![](pics/USD.png)                                | ![](pics/EUR.png)                                | ![](pics/CNY.png)                                |

## License

WebP Server is under the GPLv3. See the [LICENSE](./LICENSE) file for details.

