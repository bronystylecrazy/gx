package ws

import (
	"log"
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"go.uber.org/zap"
)

type Websocket struct {
	Url    string
	Logger *zap.Logger
}

func New(option ...Option) fiber.Handler {
	cfg := &Websocket{
		Url:    "/ws",
		Logger: zap.NewNop(), // Default to no-op logger
	}

	for _, opt := range option {
		opt(cfg)
	}

	u, err := url.Parse(cfg.Url)
	if err != nil {
		log.Fatal("Failed to parse url", zap.Error(err))
	}

	wsProxy := UseWebSocketProxyMiddleware(cfg.Logger, u)

	return adaptor.HTTPHandlerFunc(wsProxy.ServeHTTP)
}
