<p align="center">
	<img src="./pics/webp_server.png"/>
</p>
<img src="https://api.travis-ci.org/webp-sh/webp_server_go.svg?branch=master"/>

After the [n0vad3v/webp_server](https://github.com/n0vad3v/webp_server), I decide to rewrite the whole program with Go, as there will be no more `npm install`s or `docker-compose`s.

This is a Server based on Golang, which allows you to serve WebP images on the fly. 
It will convert `jpg,jpeg,png` files by default, this can be customized by editing the `config.json`.. 

> e.g When you visit `https://a.com/1.jpg`，it will serve as `image/webp` without changing the URL.
>
> For Safari and Opera users, the original image will be used.

## Compare to [n0vad3v/webp_server](https://github.com/n0vad3v/webp_server)

### Size

* `webp_server` with `node_modules`: 43M
* `webp-server(go)` single binary: 15M

### Performance

It's basically between `ExpressJS` and `Fiber`, much faster than the `http` package of course.

### Convenience

* `webp_server`: Clone the repo -> `npm install` -> run with `pm2`
* `webp-server(go)`: Download a single binary -> Run


## General Usage Steps
Regarding the `IMG_PATH` section in `config.json`. 
If you are serving images at `https://example.com/pics/tsuki.jpg` and 
your files are at `/var/www/image/pics/tsuki.jpg`, then `IMG_PATH` shall be `/var/www/image`.

## 1. Download or build the binary
Download the `webp-server` from [release](https://github.com/n0vad3v/webp_server_go/releases) page.

Wanna  build your own binary? Check out [build](#build-your-own-binaries) section

## 2. config file
Create a `config.json` as follows to face your need, default convert quality is 80%.
```json
{
	"HOST": "127.0.0.1",
	"PORT": "3333",
	"QUALITY": "80",
	"IMG_PATH": "/path/to/pics",
	"ALLOWED_TYPES": ["jpg","png","jpeg"]
}
```
## 3. Run
Run the binary like this: `./webp-server --config /path/to/config.json`, for prefetch mode, add `--prefech`.

> Prefetch mode will pre-convert all the images in the `IMG_PATH` on initial start up.

### screen or tmux
Use `screen` or `tmux` to avoid being terminated. Let's take `screen` for example
```
screen -S webp
./webp-server --config /path/to/config.json
```
(Use Ctrl-A-D to detach the `screen` with `webp-server` running.)
### systemd
Don't worry, we've got you covered!

Download `webp-server` to `/opt/webps/webp-server`, and create a config file to `/opt/webps/config.json`, then,

```shell script
cp webps.service /lib/systemd/system/
systemctl daemon-reload
systemctl enable webps.service
systemctl start webps.service
```
## 4. Nginx proxy_pass
Let Nginx to `proxy_pass http://localhost:3333/;`, and your webp-server is on-the-fly
### WordPress example
```
location ^~ /wp-content/uploads/ {
        proxy_pass http://127.0.0.1:3333;
}
```
## Build your own binaries
Install latest version of golang, enable go module, clone the repo, and then...
```shell script
make
```
**Due to the limitations of webp module, you can't cross compile this tool. 
But the binary will work instantly on your platform and arch**

## TODO
- [x] This version doesn't support header-based-output, which means Safari users will not see the converted `webp` images, this should be fixed in later releases.
- [ ] Multi platform support.
- [x] A better way to supervise the program.
- [ ] Get rid of render-blocking effect on first render.
- [x] Prefetch on server initialization.

## Related Articles(In chronological order)

* [让站点图片加载速度更快——引入 WebP Server 无缝转换图片为 WebP](https://nova.moe/re-introduce-webp-server/)
* [记 Golang 下遇到的一回「毫无由头」的内存更改](https://await.moe/2020/02/note-about-encountered-memory-changes-for-no-reason-in-golang/)
* [WebP Server in Rust](https://await.moe/2020/02/webp-server-in-rust/)
* [个人网站无缝切换图片到 webp](https://www.bennythink.com/flying-webp.html)
* [优雅的让 Halo 支持 webp 图片输出](https://halo.run/archives/halo-and-webp)

## License

WebP Server is under the GPLv3. See the [LICENSE](./LICENSE) file for details.

