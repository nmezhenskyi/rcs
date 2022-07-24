package main

import (
	"log"

	"github.com/nmezhenskyi/rcs/internal/httpsrv"
)

func main() {
	httpServer := httpsrv.NewServer()
	log.Println("--- RCS: Start ---")

	if err := httpServer.ListenAndServe("localhost:5000"); err != nil {
		log.Fatal(err)
	}
}
