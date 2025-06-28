package gx

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type Router interface {
	Setup(app fiber.Router)
}

func AsRouter(f any) fx.Option {
	return fx.Provide(
		fx.Annotate(
			f,
			fx.As(new(Router)),
			fx.ResultTags(`group:"routes"`),
		),
	)
}
