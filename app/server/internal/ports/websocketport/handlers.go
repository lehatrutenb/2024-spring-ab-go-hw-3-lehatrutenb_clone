package websocketport

import (
	"context"
	"net/http"
	"server/external/adapters"
	"sync"

	"math/rand"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const MaxLastMsgsAmt = 10

type server struct {
	clients   map[*websocket.Conn]int
	repo      adapters.Repository
	lastMsgId int
	lg        *zap.Logger
	ctx       context.Context
	eg        *errgroup.Group
	mu        *sync.Mutex
}

func newServer(ctx context.Context, eg *errgroup.Group, repo adapters.Repository, lg *zap.Logger, mu *sync.Mutex) server {
	return server{clients: make(map[*websocket.Conn]int), repo: repo, lastMsgId: -1, lg: lg, ctx: ctx, eg: eg, mu: mu}
}

func (s *server) chatHandler(w http.ResponseWriter, r *http.Request) {
	s.lg.Info("Got new websocket connection")
	conn, e := upgrader.Upgrade(w, r, nil)
	if e != nil {
		s.lg.Error("Failed to upgrade new connection", zap.Error(e))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	s.mu.Lock()
	s.clients[conn] = int(rand.Int31())
	s.mu.Unlock()
	defer delete(s.clients, conn)

	s.writeLastMessagesToNewbie(conn, MaxLastMsgsAmt)
	e = s.recieveMessages(conn)
	if e != nil && e != ErrorClosedConnection {
		s.lg.Error("Failed to recive messages", zap.Error(e))
		if e != ErrorServerFailedToReadMsg {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	s.lg.Info("End handler", zap.Int("user id", s.clients[conn]))
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
