package raft

import (
	"crypto/rand"
)

func randID() []byte {
	c := 16 // prevent collisions
	b := make([]byte, c)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return b
}
