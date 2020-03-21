<p align="center">
	<img src="./pics/webp_server.png"/>
</p>
<img src="https://api.travis-ci.org/webp-sh/webp_server_go.svg?branch=master"/>

This is a Server based on Golang, which allows you to serve WebP images on the fly. 
It will convert `jpg,jpeg,png` files by default, this can be customized by editing the `config.json`.. 
* currently supported  image format: JPEG, PNG, BMP, GIF(static image for now)


> e.g When you visit `https://a.com/1.jpg`，it will serve as `image/webp` without changing the URL.
>
> For Safari and Opera users, the original image will be used.


## Simple Usage Steps

### 1. Download or build the binary
Download the `webp-server` from [release](https://github.com/n0vad3v/webp_server_go/releases) page.

### 2. Dump config file

```
./webp-server -dump-config > config.json
```

The default `config.json` may look like this.
```json
{
	"HOST": "127.0.0.1",
	"PORT": "3333",
	"QUALITY": "80",
	"IMG_PATH": "/path/to/pics",
	"EXHAUST_PATH": "/path/to/exhaust",
	"ALLOWED_TYPES": ["jpg","png","jpeg"]
}
```

#### Config Example

In the following example, the image path and website URL.

| Image Path                            | Website Path                         |
| ------------------------------------- | ------------------------------------ |
| `/var/www/img.webp.sh/path/tsuki.jpg` | `https://img.webp.sh/path/tsuki.jpg` |

The `config.json` should be like:

| IMG_PATH               |
| ---------------------- |
| `/var/www/img.webp.sh` |


`EXHAUST_PATH` is cache folder for output `webp` images, with `EXHAUST_PATH` set to `/var/cache/webp` 
in the example above, your `webp` image will be saved at `/var/cache/webp/pics/tsuki.jpg.1582558990.webp`.

### 3. Run

```
./webp-server --config=/path/to/config.json
```

### 4. Nginx proxy_pass
Let Nginx to `proxy_pass http://localhost:3333/;`, and your webp-server is on-the-fly.

## Advanced Usage

For supervisor, Docker sections, please read our documentation at [https://webp.sh/docs/](https://webp.sh/docs/)

## Related Articles(In chronological order)

* [让站点图片加载速度更快——引入 WebP Server 无缝转换图片为 WebP](https://nova.moe/re-introduce-webp-server/)
* [记 Golang 下遇到的一回「毫无由头」的内存更改](https://await.moe/2020/02/note-about-encountered-memory-changes-for-no-reason-in-golang/)
* [WebP Server in Rust](https://await.moe/2020/02/webp-server-in-rust/)
* [个人网站无缝切换图片到 webp](https://www.bennythink.com/flying-webp.html)
* [优雅的让 Halo 支持 webp 图片输出](https://halo.run/archives/halo-and-webp)
* [让图片飞起来 oh-my-webp.sh ！](https://blog.502.li/oh-my-webpsh.html)

## License

WebP Server is under the GPLv3. See the [LICENSE](./LICENSE) file for details.

