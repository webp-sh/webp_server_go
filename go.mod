module webp_server_go

go 1.15

require (
	github.com/Kagami/go-avif v0.1.0
	github.com/chai2010/webp v1.1.0
	github.com/gofiber/fiber/v2 v2.4.0
	github.com/h2non/filetype v1.1.3
	github.com/schollz/progressbar/v3 v3.12.0
	github.com/sirupsen/logrus v1.9.0
	github.com/staktrace/go-update v0.0.0-20210525161054-fc019945f9a2
	github.com/stretchr/testify v1.8.1
	github.com/valyala/fasthttp v1.41.0
	golang.org/x/image v0.0.0-20200119044424-58c23975cae1
)

replace (
	github.com/chai2010/webp v1.1.0 => github.com/webp-sh/webp v1.2.0
	github.com/gofiber/fiber/v2 v2.4.0 => github.com/webp-sh/fiber/v2 v2.4.0
)
