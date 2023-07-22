package main

import (
	"log"

	"github.com/ianwesleyarmstrong/distributed-services-with-go-pants/internal/server"
)

func main() {
	srv := server.NewHTTPServer(":8080")
	log.Fatal(srv.ListenAndServe())
}
