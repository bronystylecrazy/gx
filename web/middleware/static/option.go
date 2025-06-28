package static

import (
	"go.uber.org/zap"
)

type Option = func(*Static)

var WithRelativeDir = func(relativeDir string) Option {
	return func(s *Static) {
		s.relativeDir = relativeDir
	}
}

var WithLogger = func(logger *zap.Logger) Option {
	return func(s *Static) {
		s.log = logger
	}
}
