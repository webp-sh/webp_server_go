<p align="center">
	<img src="./pics/webp_server.png"/>
</p>

**THIS PROJECT IS UNDER DEVELOPMENT, DON'T USE IT ON PRODUCTION ENVIRONMENT.**

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

* `webp_server`: Clone -> `npm install` -> run with `pm2`
* `webp-server(go)`: Download -> Run

## Usage

Regarding the `IMG_PATH` section in `config.json`. 
If you are serving images at `https://example.com/pics/tsuki.jpg` and 
your files are at `/var/www/image/pics/tsuki.jpg`, then `IMG_PATH` shall be `/var/www/image`.

1. Edit the `config.json` to face your need, default convert quality is 80%.
2. Run the binary like this: `./webp-server --config /path/to/config.json`, use `screen` or `tmux` to hold it currently.
3. Let Nginx to `proxy_pass http://localhost:3333/;`

## TODO
- [x] This version doesn't support header-based-output, which means Safari users will not see the converted `webp` images, this should be fixed in later releases.
- [ ] Multi platform support.

## build your own binary
Install golang, enable go module, and then...
```shell script
go get github.com/gofiber/fiber
go get github.com/chai2010/webp
go build webp_server.go
```
**Due to the limitations of webp module, you can't cross compile this tool.**