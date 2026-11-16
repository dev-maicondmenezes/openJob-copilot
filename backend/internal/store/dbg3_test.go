package store

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestDebug4(t *testing.T) {
	repo := openTestStore(t)
	ctx := context.Background()
	id, _ := repo.InsertNotification(ctx, Notification{Type: "test", Title: "x"})
	repo.MarkNotificationRead(ctx, id)
	var tv, raw any
	row := repo.db.QueryRow(`SELECT typeof(read_at), read_at FROM notifications WHERE id=?`, id)
	row.Scan(&tv, &raw)
	t.Log("typeof:", tv, "raw:", fmt.Sprintf("%v|%T", raw, raw))
	n, _ := repo.GetNotification(ctx, id)
	t.Log("model ReadAt:", n.ReadAt)
	_ = time.Now()
}
