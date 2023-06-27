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
It will convert `jpg,jpeg,png` files by default, this can be customized by editing the `config.json`..

* currently supported image format: JPEG, PNG, BMP, GIF

> e.g When you visit `https://your.website/pics/tsuki.jpg`，it will serve as `image/webp` format without changing the
> URL.
>
> ~~For Safari and Opera users, the original image will be used.~~
> We've now supported Safari/Chrome/Firefox on iOS 14/iPadOS 14

## Docker Usage steps

Please refer to [Docker | WebP Server Documentation](https://docs.webp.sh/usage/docker/) for more info.

## Simple Usage Steps(with Binary)

> Note: There is a potential memory leak problem with this server and remains unsolved, we recommend using Docker to
> mitigate this problem.
> Related discussion: https://github.com/webp-sh/webp_server_go/issues/75

### 1. Prepare the environment

#### If you are using version after 0.6.0

> Install `libvips` on your machine, minimum version 8.10.
> If your distro doesn't have a recent enough version, you can install it from source.
> We have an example for [CentOS 7](https://github.com/webp-sh/libvips/blob/master/build-full.sh)
>
> More info about [libvips](https://github.com/davidbyttow/govips)
>
> * Ubuntu `apt install --no-install-recommends libvips-dev`
> * macOS `brew install vips pkg-config`

#### If you are using version before 0.6.0

> If you'd like to run binary directly on your machine, you need to install `libaom`:
>
> `libaom` is for AVIF support, you can install it by `apt install libaom-dev` on Ubuntu, `yum install libaom-devel` on CentOS.
>
> Without this library, you may encounter error like this: `libaom.so.3: cannot open shared object file: No such file or directory`
>
> If you are using Intel Mac, you can install it by `brew install aom`
>
> If you are using Apple Silicon, you need to `brew install aom && export CPATH=/opt/homebrew/opt/aom/include/;LIBRARY_PATH=/opt/homebrew/opt/aom/lib/`, more references can be found at [在M1 Mac下开发WebP Server Go | 土豆不好吃](https://dmesg.app/m1-aom.html).
>
> If you don't like to hassle around with your system, so do us, why not have a try using Docker? >> [Docker | WebP Server Documentation](https://docs.webp.sh/usage/docker/)

### 2. Download the binary

Download the `webp-server-linux-amd64` from [Releases](https://github.com/webp-sh/webp_server_go/releases) page.

### 3. Dump config file

```
./webp-server-linux-amd64 -dump-config > config.json
```

The default `config.json` may look like this.
```json
{
  "HOST": "127.0.0.1",
  "PORT": "3333",
  "QUALITY": "80",
  "IMG_PATH": "/path/to/pics",
  "EXHAUST_PATH": "/path/to/exhaust",
  "ALLOWED_TYPES": ["jpg","png","jpeg","bmp","gif"],
  "ENABLE_AVIF": false,
  "ENABLE_EXTRA_PARAMS": false
}
```

> `ENABLE_AVIF` means AVIF support, it's disabled by default as converting images to AVIF is CPU consuming.
>
> `ENABLE_EXTRA_PARAMS` means whether to enable Extra Parameters, basically it allows you to do some transform on images like `https://img.webp.sh/path/tsuki.jpg?width=20`, you can find more info on [Extra Parameters](https://docs.webp.sh/usage/extra-params/) page.


#### Config Example

In the following example, the image path and website URL.

| Image Path                            | Website Path                         |
| ------------------------------------- | ------------------------------------ |
| `/var/www/img.webp.sh/path/tsuki.jpg` | `https://img.webp.sh/path/tsuki.jpg` |

The `IMG_PATH` inside `config.json` should be like:

| IMG_PATH               |
| ---------------------- |
| `/var/www/img.webp.sh` |


`EXHAUST_PATH` is cache folder for output `webp` images, with `EXHAUST_PATH` set to `/var/cache/webp` 
in the example above, your `webp` image will be saved at `/var/cache/webp/pics/tsuki.jpg.1582558990.webp`.

### 3. Run

```
./webp-server-linux-amd64 --config=/path/to/config.json
```

### 4. Nginx proxy_pass

Let Nginx to `proxy_pass http://localhost:3333/;`, and your WebP Server is on-the-fly.

## Advanced Usage

For supervisor, Docker sections or detailed Nginx configuration, please read our documentation at [https://docs.webp.sh/](https://docs.webp.sh/)

## Support us

If you find this project useful, please consider supporting
us by [becoming a sponsor](https://github.com/sponsors/webp-sh) or Stripe

| USD(Card, Apple Pay and Google Pay)              | EUR(Card, Apple Pay and Google Pay)              | CNY(Card, Apple Pay, Google Pay and Alipay)      |
|--------------------------------------------------|--------------------------------------------------|--------------------------------------------------|
| [USD](https://donate.stripe.com/4gwfZn2RDgag0bScMN) | [EUR](https://donate.stripe.com/28odRfgItgage2IfZ0) | [CNY](https://donate.stripe.com/00geVj8bX3nuf6MeUU) |
| ![](pics/USD.png)                                | ![](pics/EUR.png)                                | ![](pics/CNY.png)                                |

## License

WebP Server is under the GPLv3. See the [LICENSE](./LICENSE) file for details.

