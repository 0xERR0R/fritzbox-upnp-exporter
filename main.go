package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	log "github.com/sirupsen/logrus"
)

func main() {
	var config Config

	err := parse(&config)
	if err != nil {
		log.Fatalf("Could not parse config: %v\n", err)
	}

	prometheus.MustRegister(newFritzBoxCollector(&config))

	log.Info("Server is starting...")

	router := http.NewServeMux()
	router.Handle("/metrics", promhttp.Handler())
	router.HandleFunc("/all", func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Add("Content-Type", "text/plain")
		uPnPClient := NewUPnPClient(
			&config,
			make(map[string][]string),
		)
		values := uPnPClient.Execute()
		fmt.Fprintf(rw, "service:::action/variable    =    value")
		for _, v := range values {
			fmt.Fprintf(rw, "%s:::%s/%s   =   %s\n", v.serviceType, v.actionName, v.variable, v.value)
		}
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%v", 8080),
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	done := make(chan bool)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		<-quit
		log.Info("Server is shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	log.Info("Server is ready to handle requests at :", 8080)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on %d: %v\n", 8080, err)
	}

	<-done
	log.Info("Server stopped")
}
