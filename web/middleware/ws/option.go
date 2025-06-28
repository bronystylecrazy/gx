package ws

import "go.uber.org/zap"

type Option = func(*Websocket)

var WithWsEndpoint = func(endpoint string) Option {
	return func(w *Websocket) {
		w.Url = endpoint
	}
}

var WithLogger = func(logger *zap.Logger) Option {
	return func(s *Websocket) {
		s.Logger = logger
	}
}
