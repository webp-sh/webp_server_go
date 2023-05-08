module webp_server_go

go 1.20

require (
	github.com/davidbyttow/govips v0.0.0-20201026223743-b1b72c7305d9
	github.com/davidbyttow/govips/v2 v2.13.0
	github.com/gofiber/fiber/v2 v2.4.0
	github.com/h2non/filetype v1.1.3
	github.com/schollz/progressbar/v3 v3.13.1
	github.com/sirupsen/logrus v1.9.0
	github.com/staktrace/go-update v0.0.0-20210525161054-fc019945f9a2
	github.com/stretchr/testify v1.8.2
	github.com/valyala/fasthttp v1.47.0
	golang.org/x/image v0.7.0
)

require (
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/klauspost/compress v1.16.3 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rivo/uniseg v0.4.3 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/term v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/chai2010/webp v1.1.0 => github.com/webp-sh/webp v1.2.0
	github.com/gofiber/fiber/v2 v2.4.0 => github.com/webp-sh/fiber/v2 v2.4.0
)
