package main

import (
	"log"

	"github.com/nmezhenskyi/rcs/internal/server"
)

func main() {
	httpServer := server.NewHTTPServer()
	log.Println("--- RCS: Start ---")

	if err := httpServer.ListenAndServe("localhost:5000"); err != nil {
		log.Fatal(err)
	}
}
