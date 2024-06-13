package websocketport

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"server/external/adapters/storagerepo"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var ErrorServerShutDown error = errors.New("server is shutting down")

func RunServer(addr string, rAddrs map[string]string) {
	var httpSrv http.Server
	httpSrv.Addr = addr
	lg, e := zap.NewDevelopment()
	if e != nil {
		log.Fatal("Failed to init logger")
	}

	eg, ctx := errgroup.WithContext(context.Background())
	server := newServer(ctx, eg, storagerepo.NewRepo(ctx, rAddrs, lg), lg, &sync.Mutex{})
	http.HandleFunc("/", server.chatHandler)
	server.waitForMessages()

	sigQuit := make(chan os.Signal, 2)
	signal.Notify(sigQuit, syscall.SIGINT, syscall.SIGTERM)

	eg.Go(func() error {
		s := <-sigQuit
		lg.Warn("Shutting server down", zap.Any("signal", s))
		server.closeConns()

		if e := httpSrv.Shutdown(ctx); e != nil {
			server.lg.Info("http server shutdown", zap.Error(e))
		}

		if e = server.repo.CloseRepo(); e != nil {
			server.lg.Error("Failed to close repo", zap.Error(e))
		}
		return ErrorServerShutDown
	})

	eg.Go(func() error {
		return httpSrv.ListenAndServe()
	})

	if e = eg.Wait(); e != nil && e != ErrorServerShutDown && e != http.ErrServerClosed {
		server.lg.Fatal("Server stoppend listening unintentionally", zap.Error(e))
		return
	}
	server.lg.Info("Server shutted down succesfully", zap.Error(e))
}
