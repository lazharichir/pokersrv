package main

import (
	"fmt"
	"log"

	"github.com/lazharichir/poker/server"
)

func main() {
	fmt.Println("Starting Unique Poker Game Backend...")

	s := server.NewServer()
	err := s.Start("7777")

	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
