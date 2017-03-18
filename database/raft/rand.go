package raft

import (
	"crypto/rand"
)

func randID() []byte {
	b := make([]byte, 16) // 16 is long enough to prevent collisions
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return b
}
