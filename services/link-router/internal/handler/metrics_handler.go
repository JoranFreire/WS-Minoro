package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	fasthttpadaptor "github.com/valyala/fasthttp/fasthttpadaptor"
)

func MetricsHandler() fiber.Handler {
	h := promhttp.Handler()
	fh := fasthttpadaptor.NewFastHTTPHandler(h)
	return func(c *fiber.Ctx) error {
		fh(c.Context())
		return nil
	}
}
