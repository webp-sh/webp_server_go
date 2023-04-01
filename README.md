<p align="center">
	<img src="./pics/webp_server.png"/>
</p>

[![CI](https://github.com/webp-sh/webp_server_go/actions/workflows/CI.yaml/badge.svg)](https://github.com/webp-sh/webp_server_go/actions/workflows/CI.yaml)
[![build docker image](https://github.com/webp-sh/webp_server_go/actions/workflows/release_binary.yaml/badge.svg)](https://github.com/webp-sh/webp_server_go/actions/workflows/release_binary.yaml)
[![Release WebP Server Go Binaries](https://github.com/webp-sh/webp_server_go/actions/workflows/release_docker_image.yaml/badge.svg)](https://github.com/webp-sh/webp_server_go/actions/workflows/release_docker_image.yaml)
[![codecov](https://codecov.io/gh/webp-sh/webp_server_go/branch/master/graph/badge.svg?token=VR3BMZME65)](https://codecov.io/gh/webp-sh/webp_server_go)
![Docker Pulls](https://img.shields.io/docker/pulls/webpsh/webp-server-go?style=plastic)

[Documentation](https://docs.webp.sh/) | [Website](https://webp.sh/)

This Golang-based server allows you to serve WebP images on the fly, converting jpg, jpeg, png files by default.

The conversion options can be customized by editing the `config.json` file.

* The server currently supports the following image formats: JPEG, PNG, BMP, and GIF (static image for now).

> For example, when you visit the URL `https://your.website/pics/tsuki.jpg`,
> the image will be served in `image/webp` format without altering the URL.
>
> Previously, Safari and Opera users would receive the original image,
> but now we also support Safari, Chrome, and Firefox on iOS 14 and iPadOS 14.

## Simple Usage Steps(with Binary)

### 1. Prepare the environment

To run the binary directly on your machine, you need to install libaom:

To enable AVIF support, `libaom` needs to be installed.

If `libaom` is not installed, you may encounter an error message like this:
`libaom.so.3: cannot open shared object file: No such file or directory.`

* If you're using Ubuntu, you can install it by running `apt install libaom-dev`.
* If you're using CentOS, you can install it by running `yum install libaom-devel`.
* If you are using an Intel-based Mac, you can install the aom library by running `brew install aom` in your terminal.
* However, if you are using an Apple Silicon-based Mac,
  you need to run the following commands to install the aom library and set the environment variables for the library:

```shell
brew install aom
export CPATH=/opt/homebrew/opt/aom/include/
export LIBRARY_PATH=/opt/homebrew/opt/aom/lib/
```

For more information, you can refer to this
guide:  [在M1 Mac下开发WebP Server Go | 土豆不好吃](https://dmesg.app/m1-aom.html).

If you prefer to avoid dealing with system dependencies, you can try running the server using
Docker. [Docker | WebP Server Documentation](https://docs.webp.sh/usage/docker/)

### 2. Download the binary

Download the `webp-server-linux-amd64` from [Releases](https://github.com/webp-sh/webp_server_go/releases) page.

### 3. Dump config file

```
./webp-server-linux-amd64 -dump-config > config.json
```

Here's an example of what the `config.json` file may look like by default:

```json
{
  "HOST": "127.0.0.1",
  "PORT": "3333",
  "QUALITY": "80",
  "IMG_PATH": "/path/to/pics",
  "EXHAUST_PATH": "/path/to/exhaust",
  "ALLOWED_TYPES": [
    "jpg",
    "png",
    "jpeg",
    "bmp"
  ],
  "ENABLE_AVIF": false
}
```

> By default, AVIF support is disabled in the server because the process of converting images to AVIF format is
> CPU-intensive.

#### Config Example

The table below shows an example of the image path and corresponding website URL.

| Image Path                            | Website Path                         |
|---------------------------------------|--------------------------------------|
| `/var/www/img.webp.sh/path/tsuki.jpg` | `https://img.webp.sh/path/tsuki.jpg` |

The `IMG_PATH` in the `config.json` file should be set as follows:

| IMG_PATH               |
|------------------------|
| `/var/www/img.webp.sh` |

The `EXHAUST_PATH` is the cache folder where the output webp images are stored.

In the example above, if `EXHAUST_PATH` is set to `/var/cache/webp`,
your webp image will be saved at `/var/cache/webp/pics/tsuki.jpg.1582558990.webp`.

### 3. Run

```
./webp-server-linux-amd64 --config=/path/to/config.json
```

### 4. Nginx proxy_pass

You can make your WebP server on-the-fly by configuring Nginx to `proxy_pass http://localhost:3333/;`.

## Advanced Usage

For more information on configuring the WebP Server with supervisor, Docker, or detailed Nginx settings,
please refer to our documentation located at: [https://docs.webp.sh/](https://docs.webp.sh/)

## Support us

Consider supporting this project
if you find it useful by becoming a sponsor through [becoming a sponsor](https://github.com/sponsors/webp-sh) or using
Stripe.

| USD(Card, Apple Pay and Google Pay)              | SEK(Card, Apple Pay and Google Pay)              | CNY(Card, Apple Pay, Google Pay and Alipay)      |
|--------------------------------------------------|--------------------------------------------------|--------------------------------------------------|
| [USD](https://buy.stripe.com/cN203sdZB98RevC3cd) | [SEK](https://buy.stripe.com/bIYbMa9JletbevCaEE) | [CNY](https://buy.stripe.com/dR67vU4p13Ox73a6oq) |
| ![](pics/USD.png)                                | ![](pics/SEK.png)                                | ![](pics/CNY.png)                                |

## License

WebP Server Go is released under the GPLv3 license. See the [LICENSE](./LICENSE) file for details.

