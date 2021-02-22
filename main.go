package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var (
	port = "8080"
	addr = "127.0.0.1"
)

func init() {
	log.SetLevel(log.DebugLevel)
	addr = validIP()
}

func validIP() string {
	if v := os.Getenv("LISTEN_ADDR"); v == "" {
		// ADDR not filled, return default
		return addr
	}

	if net.ParseIP(os.Getenv("LISTEN_ADDR")) == nil {
		log.Fatalln("invalid $LISTEN_ADDR")
	}

	return os.Getenv("LISTEN_ADDR")
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/calendar/{id}", RenderICalHandler).Methods(http.MethodGet)
	r.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "OK")
	}).Methods(http.MethodGet)

	// Log global router access logs
	loggedRouter := handlers.LoggingHandler(os.Stdout, r)

	srv := &http.Server{
		Addr:         addr + ":" + port,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      loggedRouter,
	}

	go func() {
		log.Printf("listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Println("shutting down")
	os.Exit(0)
}
