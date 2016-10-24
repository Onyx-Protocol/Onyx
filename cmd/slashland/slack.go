package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func sayf(format string, args ...interface{}) {
	say(fmt.Sprintf(format, args...))
}

// say sends a message to the configured slack channel
func say(args ...interface{}) {
	message := fmt.Sprintln(args...)
	body, err := json.Marshal(map[string]string{"text": message})
	if err != nil {
		panic(err)
	}
	r, err := http.Post(*postURL, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Println(err)
	}
	defer r.Body.Close()
	if r.StatusCode/100 != 2 {
		log.Println("bad status", r.Status)
	}
}
