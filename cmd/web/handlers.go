package main

import (
	"log"
	"net/http"
)

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", serveHome)
	mux.HandleFunc("/wait", serveWaitingRoom)
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("Hello world!"))
	if err != nil {
		log.Println(err)
	}
}

func serveWaitingRoom(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("Waiting room!"))
	if err != nil {
		log.Println(err)
	}
}
