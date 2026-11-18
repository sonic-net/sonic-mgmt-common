package db

import "testing"

func TestRedisExpiredEventIsLogicalDelete(t *testing.T) {
	database := &DB{}
	if got := database.redisPayload2sEvent("expired"); got != SEventDel {
		t.Fatalf("expired event mapped to %v, want %v", got, SEventDel)
	}
}
