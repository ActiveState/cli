package server

import (
	"log"
	"net/http"
)

var srv http.Server

// Up runs the http server, serving up files under ./data
func Up() {
	srv := &http.Server{Addr: ":8100"}
	http.Handle("/", http.FileServer(http.Dir("./data")))

	if err := srv.ListenAndServe(); err != nil {
		// cannot panic, because this probably is an intentional close
		log.Printf("Httpserver: ListenAndServe() error: %s", err)
	}
}

// Down takes down the server
func Down() {
	err := srv.Shutdown(nil)
	if err != nil {
		panic(err) // failure/timeout shutting down the server gracefully
	}
}
