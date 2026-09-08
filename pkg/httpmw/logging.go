package httpmw

import (
	"net/http"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func Logging(logger *zap.Logger, quietPaths ...string) func(http.Handler) http.Handler {
	quiet := make(map[string]struct{}, len(quietPaths))
	for _, path := range quietPaths {
		quiet[path] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()

			next.ServeHTTP(rec, r)

			if _, skip := quiet[r.URL.Path]; skip && rec.status < http.StatusBadRequest {
				return
			}

			logger.Log(levelForStatus(rec.status), "http request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr),
				zap.Int("status", rec.status),
				zap.Duration("duration", time.Since(start)),
			)
		})
	}
}

func levelForStatus(status int) zapcore.Level {
	switch {
	case status >= http.StatusInternalServerError:
		return zapcore.ErrorLevel
	case status >= http.StatusBadRequest:
		return zapcore.WarnLevel
	default:
		return zapcore.InfoLevel
	}
}
