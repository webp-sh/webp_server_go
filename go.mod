module webp_server_go

go 1.13

require (
	github.com/chai2010/webp v1.1.0
	github.com/gofiber/fiber/v2 v2.1.4
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.3.0
	golang.org/x/image v0.0.0-20200119044424-58c23975cae1
)

replace (
	github.com/gofiber/fiber/v2 v2.1.4  => github.com/webp-sh/fiber/v2 v2.1.4
	github.com/chai2010/webp v1.1.0  => github.com/webp-sh/webp v1.1.1
)