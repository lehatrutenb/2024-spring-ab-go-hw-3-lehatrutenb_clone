package postgresrepo

import (
	"context"
	"server/external/message"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type PostgresRepo struct {
	conn     *pgx.Conn
	lg       *zap.Logger
	ctx      context.Context
	lMsgTmSt int64
}

func NewRepo(dbAddr string, ctx context.Context, lg *zap.Logger) *PostgresRepo {
	conn, err := pgx.Connect(context.Background(), dbAddr)
	if err != nil {
		lg.Fatal("Failed to connect to postgres repo", zap.Error(err))
	}
	return &PostgresRepo{conn: conn, lg: lg, ctx: ctx, lMsgTmSt: -1}
}

func scanAllMessages(rows pgx.Rows) ([]message.Message, []int64, error) {
	msgs := make([]message.Message, 0)
	tStamps := make([]int64, 0)
	for rows.Next() {
		var msg message.Message
		var uId int
		var tStamp int64
		if e := rows.Scan(&uId, &msg.User, &msg.Text, &tStamp); e != nil {
			return []message.Message{}, []int64{}, e
		}
		msg.SetUserId(uId)
		msgs = append(msgs, msg)
		tStamps = append(tStamps, tStamp)
	}

	return msgs, tStamps, nil
}

const GetNewerMessagesQuery = `SELECT userid, username, text, timestamp FROM messages WHERE timestamp > $1`

// TODO messages with equal timestamp is nearly impossible, and I don't really know what to do if we have 3 such messages - now we will loose them
func (pr *PostgresRepo) GetNewerMessages() ([]message.Message, error) {
	rows, e := pr.conn.Query(pr.ctx, GetNewerMessagesQuery, pr.lMsgTmSt)
	if e != nil {
		pr.lg.Error("Failed to query messages from repo", zap.Error(e))
		return []message.Message{}, e
	}

	defer rows.Close()

	msgs, ts, e := scanAllMessages(rows)
	if e != nil {
		pr.lg.Error("Failed to scan messages from repo", zap.Error(e))
		return []message.Message{}, e
	}
	if len(ts) != 0 {
		pr.lMsgTmSt = ts[len(ts)-1]
	}
	return msgs, nil
}

const GetLastMessagesQuery = `SELECT userid, username, text, timestamp FROM messages ORDER BY timestamp DESC LIMIT $1;`

func (pr *PostgresRepo) GetLastKMessages(k int) ([]message.Message, error) {
	rows, e := pr.conn.Query(pr.ctx, GetLastMessagesQuery, k)
	if e != nil {
		pr.lg.Error("Failed to query messages from repo", zap.Error(e), zap.Int("Msg amt", k))
		return []message.Message{}, e
	}

	defer rows.Close()

	ms, _, e := scanAllMessages(rows)
	if e != nil {
		pr.lg.Error("Failed to scan messages from repo", zap.Error(e))
		return []message.Message{}, e
	}
	return ms, nil
}

const AddMessageQuery = `INSERT INTO messages (username, text, chatid, userid, timestamp) VALUES ($1, $2, $3, $4, $5)`

func (pr *PostgresRepo) AddMessage(m message.Message) error {
	pr.lg.Debug("Add message", zap.Int("user id ", m.GetUserId()), zap.Int("chat id", m.GetChatId()))
	_, e := pr.conn.Exec(context.Background(), AddMessageQuery, m.User, m.Text, m.GetChatId(), m.GetUserId(), time.Now().UnixMilli())
	if e != nil {
		pr.lg.Error("Failed to add message to repo", zap.Error(e), zap.Int("user id", m.GetUserId()), zap.Int("chat id", m.GetChatId()))
		return e
	}
	return nil
}

func (pr *PostgresRepo) SetLastMessageTimeStamp(timeSt int64) {
	pr.lMsgTmSt = timeSt
}

func (pr *PostgresRepo) GetLastMessageTimeStamp() int64 {
	return pr.lMsgTmSt
}

func (pr *PostgresRepo) CloseRepo() error {
	return pr.conn.Close(pr.ctx)
}
