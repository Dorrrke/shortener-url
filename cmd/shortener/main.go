// Модуль main - основная точка входа в систему.
// В пакете происходит подключение к базе данных, если имеется ссылка для подключения, создание storage, и инициализация logger и server.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"net/http/pprof"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/errgroup"

	"github.com/Dorrrke/shortener-url/internal/config"
	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/Dorrrke/shortener-url/pkg/server"
	"github.com/Dorrrke/shortener-url/pkg/storage"
)

// FilePath — константа с названием файла для хранения данных при отсутствии подключения к бд.
const FilePath string = "short-url-db.json"

// Глобальные переменные для вывода при запуске.
var (
	// buildVersion - версия сборки.
	buildVersion string
	// buildDate - дата сборки.
	buildDate string
	// buildCommit - комментарии к сборке.
	buildCommit string
)

func main() {
	if buildVersion == "" {
		buildVersion = "N/A"
	} else {
		fmt.Printf("Build version: %s\n", buildVersion)
	}

	if buildDate == "" {
		buildDate = "N/A"
	} else {
		fmt.Printf("Build date: %s\n", buildDate)
	}

	if buildCommit == "" {
		buildCommit = "N/A"
	} else {
		fmt.Printf("Build commit: %s\n", buildCommit)
	}

	if err := logger.Initialize(zap.InfoLevel.String()); err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

		<-c
		cancel()
	}()

	var stor storage.Storage
	appCfg := config.MustLoad()
	logger.Log.Info("Server config", zap.Any("cfg", appCfg))
	if appCfg.DatabaseDsn != "" {
		dbConn := initDB(appCfg.DatabaseDsn)
		stor = &storage.DBStorage{DB: dbConn}
	} else {
		stor = &storage.MemStorage{URLMap: make(map[string]string)}
	}

	serverAPI := server.New(stor, appCfg)
	if err := serverAPI.RestorStorage(); err != nil {
		logger.Log.Error("Error restor storage: ", zap.Error(err))
	}

	server := &http.Server{}

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return run(*serverAPI, server)
	})
	g.Go(func() error {
		<-gCtx.Done()
		return stopService(server)
	})

	if err := g.Wait(); err != nil {
		logger.Log.Error("server stoped", zap.String("exit reason", err.Error()))
	}
}

func run(serv server.Server, serverHTTP *http.Server) error {

	logger.Log.Info("Running server")
	r := chi.NewRouter()

	r.Route("/", func(r chi.Router) {
		r.Post("/", logger.WithLogging(server.GzipMiddleware(serv.ShortenerURLHandler)))
		r.Get("/{id}", logger.WithLogging(server.GzipMiddleware(serv.GetOriginalURLHandler)))
		r.Route("/api", func(r chi.Router) {
			r.Get("/user/urls", logger.WithLogging(server.GzipMiddleware(serv.GetAllUrls)))
			r.Delete("/user/urls", logger.WithLogging(server.GzipMiddleware(serv.DeleteURLHandler)))
			r.Route("/shorten", func(r chi.Router) {
				r.Post("/", logger.WithLogging(server.GzipMiddleware(serv.ShortenerJSONURLHandler)))
				r.Post("/batch", logger.WithLogging(server.GzipMiddleware(serv.InsertBatchHandler)))
			})
		})
		r.Get("/ping", logger.WithLogging(server.GzipMiddleware(serv.CheckDBConnectionHandler)))
	})
	r.HandleFunc("/debug/pprof", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, r.URL.Path[1:])
	})

	// Регистрация обработчиков pprof для различных типов профилирования
	r.HandleFunc("/debug/pprof/heap", pprof.Index)
	r.HandleFunc("/debug/pprof/goroutine", pprof.Index)
	r.HandleFunc("/debug/pprof/block", pprof.Index)
	r.HandleFunc("/debug/pprof/threadcreate", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

	serverHTTP.Handler = r
	if serv.Config.ServerAddress != "" {
		serverHTTP.Addr = serv.Config.ServerAddress
	} else {
		serverHTTP.Addr = ":8080"
	}
	if serv.Config.EnableHTTPS {
		manager := &autocert.Manager{
			Cache:      autocert.DirCache("cache-dir"),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist("short.ru"),
		}
		serverHTTP.TLSConfig = manager.TLSConfig()
		logger.Log.Info("Server with TLS started", zap.String("addres", serv.Config.ServerAddress))
		return serverHTTP.ListenAndServeTLS("", "")
	}
	logger.Log.Info("Server without TLS started", zap.String("addres", serv.Config.ServerAddress))
	return serverHTTP.ListenAndServe()
}

func initDB(DBAddr string) *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), DBAddr)
	if err != nil {
		logger.Log.Error("Error wile init db driver: " + err.Error())
		panic(err)
	}
	return pool

}

func stopService(serverHTTP *http.Server) error {
	serverHTTP.Shutdown(context.Background())
	logger.Log.Info("Service stop")
	return nil
}
