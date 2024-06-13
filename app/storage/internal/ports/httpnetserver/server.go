package httpnetserver

import (
	"net/http"
	"storage/internal/consumer"

	"go.uber.org/zap"
)

func RunServer(addr string, mh *consumer.MessageHandler) {
	var httpSrv http.Server
	httpSrv.Addr = addr
	s := newServer(mh)

	http.HandleFunc("/get", http.HandlerFunc(s.getNewMessagesHandler))
	http.HandleFunc("/get_newbie", http.HandlerFunc(s.getNewbieMessagesHandler))

	mh.Eg.Go(func() error {
		<-mh.Ctx.Done()
		mh.Lg.Warn("Shutting server down", zap.Any("signal", s))

		if err := httpSrv.Shutdown(mh.Ctx); err != nil {
			mh.Lg.Info("http server shutdown", zap.Error(err))
			return err
		}
		return nil
	})

	mh.Eg.Go(func() error {
		return httpSrv.ListenAndServe()
	})
	mh.Lg.Info("Server is running")
}
