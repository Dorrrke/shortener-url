// Пакет logger содержит в себе синголтон логгера zap и реализацию mw с логгированием.
package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Log - Singletone логгера.
var Log *zap.Logger = zap.NewNop()

// Initialize - функция инициализации zap.Logger.
func Initialize(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = lvl

	zl, err := cfg.Build()
	if err != nil {
		return err
	}

	Log = zl
	return nil
}

// Структуры данных для логирования запросов.
type (
	responceData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responceData *responceData
	}
)

// Дополненый метод Write для логирования запросов.
func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responceData.size += size
	return size, err
}

// Дополненый метод WriteHeader для логирования запросов.
func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responceData.status = statusCode
}

// WithLogging - middleware для логгирвоания запростов к серверу.
func WithLogging(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		responceData := &responceData{
			status: 0,
			size:   0,
		}

		lw := loggingResponseWriter{
			ResponseWriter: w,
			responceData:   responceData,
		}

		uri := r.RequestURI

		method := r.Method

		h.ServeHTTP(&lw, r)

		duration := time.Since(start)

		Log.Info("Request: ",
			zap.String("method", method),
			zap.String("uri", uri),
			zap.String("duration", duration.String()))

		Log.Info("Response: ",
			zap.Int("status", responceData.status),
			zap.Int("size", (responceData.size)))
	})
}
