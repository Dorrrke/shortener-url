package server

import (
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"strings"
)

// Доплоненный метод Write для реализации mv с жатием данных.
type gzipWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

// Создание Writer со сжатием.
func newGzipWriter(w http.ResponseWriter) *gzipWriter {
	return &gzipWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

// Реализация метода Header.
func (c *gzipWriter) Header() http.Header {
	return c.w.Header()
}

// Реализация метода Write.
func (c *gzipWriter) Write(p []byte) (int, error) {
	return c.zw.Write(p)
}

// Реализация метода WriteHeader.
func (c *gzipWriter) WriteHeader(statusCode int) {
	if statusCode < 300 || statusCode == 409 {
		c.w.Header().Set("Content-Encoding", "gzip")
	}
	c.w.WriteHeader(statusCode)
}

// Close закрывает gzip.Writer и досылает все данные из буфера.
func (c *gzipWriter) Close() error {
	return c.zw.Close()
}

// gzipReader реализует интерфейс io.ReadCloser и позволяет прозрачно для сервера декомпрессировать получаемые от клиента данные.
type gzipReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

// Метод созадния экземпляра newGzipReader.
func newGzipReader(r io.ReadCloser) (*gzipReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &gzipReader{
		r:  r,
		zr: zr,
	}, nil
}

// Hеализация метода Read.
func (c gzipReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

// Реализация метода Close.
func (c *gzipReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

// Middleware со сжатием данных.
func GzipMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			if strings.Contains(r.Header.Get("Content-Type"), "application/json") || strings.Contains(r.Header.Get("Content-Type"), "text/html") {
				if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
					h.ServeHTTP(w, r)
					return
				}
				gzipWriter := newGzipWriter(w)
				w = gzipWriter
				defer gzipWriter.Close()
				h.ServeHTTP(w, r)
				return
			}
			h.ServeHTTP(w, r)
			return
		}
		gzipReader, err := newGzipReader(r.Body)
		if err != nil {
			log.Print(err.Error())
			return
		}
		r.Body = gzipReader
		if strings.Contains(r.Header.Get("Content-Type"), "application/json") || strings.Contains(r.Header.Get("Content-Type"), "text/html") {
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				h.ServeHTTP(w, r)
				return
			}
			gzipWriter := newGzipWriter(w)
			w = gzipWriter
			defer gzipWriter.Close()
			h.ServeHTTP(w, r)
			return
		}
		h.ServeHTTP(w, r)
	}
}
