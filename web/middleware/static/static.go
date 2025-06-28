package static

import (
	"embed"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type Static struct {
	FS          embed.FS
	log         *zap.Logger
	relativeDir string
}

func New(fs embed.FS, option ...Option) *Static {
	s := &Static{
		FS:          fs,
		log:         zap.NewNop(), // Default to no-op logger
		relativeDir: "web/dist",
	}

	for _, opt := range option {
		opt(s)
	}

	return s
}

func (s *Static) Setup(app fiber.Router) {
	subbedDist, err := fs.Sub(s.FS, s.relativeDir)
	if err != nil {
		s.log.Fatal("failed to get relative directory", zap.Error(err))
	}

	app.Use(func(c *fiber.Ctx) error {
		path := c.Path()
		if path == "/" {
			path = "/index.html"
		}

		// Check gzip support
		acceptsGzip := strings.Contains(c.Get("Accept-Encoding"), "gzip")

		// Try to serve .gz version
		if acceptsGzip {
			gzPath := path + ".gz"
			if file, err := subbedDist.Open(gzPath[1:]); err == nil {
				defer file.Close()
				data, err := io.ReadAll(file)
				if err != nil {
					return err
				}

				c.Set("Content-Encoding", "gzip")
				c.Set("Vary", "Accept-Encoding")
				c.Type(filepath.Ext(path)) // get mime based on original ext
				return c.Send(data)
			}
		}

		// Try to serve normal file
		file, err := subbedDist.Open(path[1:])
		if err != nil {
			// Fallback to index.html (SPA)
			index, err := subbedDist.Open("index.html.gz")
			if err != nil {
				return fiber.ErrNotFound
			}
			defer index.Close()
			data, err := io.ReadAll(index)
			if err != nil {
				return err
			}
			c.Set("Content-Encoding", "gzip")
			c.Set("Vary", "Accept-Encoding")
			c.Type("html")
			return c.Send(data)
		}

		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			return err
		}
		c.Type(filepath.Ext(path))
		return c.Send(data)
	})
}
