<p align="center">
	<img src="./pics/webp_server.png"/>
</p>

[![CI](https://github.com/webp-sh/webp_server_go/actions/workflows/CI.yaml/badge.svg)](https://github.com/webp-sh/webp_server_go/actions/workflows/CI.yaml)
[![build docker image](https://github.com/webp-sh/webp_server_go/actions/workflows/release_binary.yaml/badge.svg)](https://github.com/webp-sh/webp_server_go/actions/workflows/release_binary.yaml)
[![Release WebP Server Go Binaries](https://github.com/webp-sh/webp_server_go/actions/workflows/release_docker_image.yaml/badge.svg)](https://github.com/webp-sh/webp_server_go/actions/workflows/release_docker_image.yaml)
[![codecov](https://codecov.io/gh/webp-sh/webp_server_go/branch/master/graph/badge.svg?token=VR3BMZME65)](https://codecov.io/gh/webp-sh/webp_server_go)
![Docker Pulls](https://img.shields.io/docker/pulls/webpsh/webp-server-go?style=plastic)

[Documentation](https://docs.webp.sh/) | [Website](https://webp.sh/)

This is a Server based on Golang, which allows you to serve WebP images on the fly.

* currently supported image format: JPEG, PNG, BMP, GIF

> e.g When you visit `https://your.website/pics/tsuki.jpg`，it will serve as `image/webp` format without changing the
> URL.
>
> ~~For Safari and Opera users, the original image will be used.~~
> We've now supported Safari/Chrome/Firefox on iOS 14/iPadOS 14

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
    environment:
      - MALLOC_ARENA_MAX=1
    volumes:
      - ./path/to/pics:/opt/pics
      - ./exhaust:/opt/exhaust
    ports:
      -  127.0.0.1:3333:3333
```

Suppose your website and image has the following pattern.

| Image Path                            | Website Path                         |
| ------------------------------------- | ------------------------------------ |
| `/var/www/img.webp.sh/path/tsuki.jpg` | `https://img.webp.sh/path/tsuki.jpg` |

Then

* `./path/to/pics` should be changed to `/var/www/img.webp.sh`
* `./exhaust` is cache folder for output images, by default it will be in `exhaust` directory alongside with `docker-compose.yml` file, if you'd like to keep cached images in another folder as , you can change  `./exhaust` to `/some/other/path/to/exhaust`

Start the container using:

```
docker-compose up -d
```

Now the server should be running on `127.0.0.1:3333`, visiting `http://127.0.0.1:3333/path/tsuki.jpg` will see the optimized version of `/var/www/img.webp.sh/path/tsuki.jpg`, you can now add reverse proxy to make it public, for example, let Nginx to `proxy_pass http://127.0.0.1:3333/;`, and your WebP Server is on-the-fly!

You can refer to [Docker | WebP Server Documentation](https://docs.webp.sh/usage/docker/) for more info, such as custom config, AVIF support etc.

## Advanced Usage

If you'd like to use with binary, please consult to [Basic Usage | WebP Server Documentation](https://docs.webp.sh/usage/basic-usage/), spoiler alert: you may encounter issues with `glibc` and some dependency libraries.

For supervisor or detailed Nginx configuration, please read our documentation at [https://docs.webp.sh/](https://docs.webp.sh/)

### Lazy mode notes

`-lazy` flag makes the server to process images in the background. It is designed to provide fastest possible responses, but optimized formats might take some seconds to be available. 

You might use this mode when you want to limit CPU and RAM resources consumption. For example, when a AVIF is enabled, the system might use up to #CPU * 1GB for the server to process #CPU concurrent AVIF image conversions.

```
# For example, with a 4CPU, 4GB RAM system running with the following parameters:
./webp-server-linux-amd64 -lazy
```

Can consume up to `1024 * 4` (Heavy/Avif conversions) =~ 4.0GB of RAM

For a minimal footprint, adapt the number of background heavy jobs (`-lazy-heavy-jobs` flag) to the amount of RAM you can dedicate to the process. 

```
# For example, with a 4CPU, 4GB RAM system running with the following parameters:
./webp-server-linux-amd64 -lazy -lazy-heavy-jobs 1
```

Can consume up to `128MB * 3` (Default/Webp parallel conversions) + `1024 * 1` (Heavy/Avif conversions) =~ 1.5GB of RAM

## WebP Cloud Services

We are currently building a new service called [WebP Cloud Services](https://webp.se/), it now has two services:

* [Public Service](https://public.webp.se)
  * GitHub Avatar/Gravater reverse proxy with WebP optimization, for example, change `https://www.gravatar.com/avatar/09eba3a443a7ea91cf818f6b27607d66` to `https://gravatar.webp.se/avatar/09eba3a443a7ea91cf818f6b27607d66` for rendering will get a smaller version of gravater, making your website faster 
  * Totally free service and currently has a large number of users, this includes, but is not limited to [CNX Software](https://medium.com/amarao/scaleway-arm-servers-50f85c4cefbe),[Indienova](https://indienova.com/en) 
* [WebP Cloud](https://docs.webp.se/webp-cloud/)
  * No need to install WebP Server Go, especially suitable for static websites.
  * Image Conversion: WebP Cloud converts images to WebP format, reducing size while maintaining quality for faster website loading.
    * Example: Original image URL (https://yyets.dmesg.app/api/user/avatar/Benny) becomes compressed URL (https://vz4w427.webp.ee/api/user/avatar/Benny).
  * Caching: WebP Cloud automatically caches served images, reducing traffic and bandwidth load on the origin server.

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

