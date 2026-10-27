// Command ari-sim runs a simulated Asterisk ARI server so the middleware and
// control panel can be demoed and tested without a PBX.
//
//	go run ./cmd/ari-sim -listen :8088 -rate 4 -speed 1 -answer 0.85
//
// Point the middleware at it with ARI_URL=http://127.0.0.1:8088/ari and
// ARI_WS_URL=ws://127.0.0.1:8088/ari/events. Inject a scripted call with:
//
//	curl -XPOST localhost:8088/sim/call -d '{"caller":"+15550100","dnis":"100","digits":"1"}'
package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"callcenter-middleware/internal/sim"
)

func main() {
	listen := flag.String("listen", ":8088", "listen address")
	app := flag.String("app", "callcenter-app", "ARI application name")
	user := flag.String("user", "callcenter-app", "ARI username")
	pass := flag.String("pass", "pbx_ari_secret_2026", "ARI password")
	rate := flag.Float64("rate", 3, "automatic inbound calls per minute (0 disables)")
	speed := flag.Float64("speed", 1, "time acceleration factor")
	answer := flag.Float64("answer", 0.85, "probability an agent phone answers")
	dnis := flag.String("dnis", "100,100,100,700", "comma separated dialed numbers for generated calls")
	digits := flag.String("digits", "11122339", "menu keys generated callers choose from")
	debug := flag.Bool("debug", false, "verbose logging")
	flag.Parse()

	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	s := sim.New(sim.Options{App: *app, Username: *user, Password: *pass, Speed: *speed, AnswerProb: *answer,
		CallsPerMinute: *rate, DNIS: strings.Split(*dnis, ","), Digits: *digits})
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	go s.RunGenerator(ctx)

	srv := &http.Server{Addr: *listen, Handler: s.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		slog.Info("ARI simulator listening", "addr", *listen, "rate_per_min", *rate, "speed", *speed)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("listen failed", "err", err)
			os.Exit(1)
		}
	}()
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				slog.Info("simulator stats", "stats", s.Stats())
			}
		}
	}()
	<-ctx.Done()
	shutdown, c := context.WithTimeout(context.Background(), 3*time.Second)
	defer c()
	_ = srv.Shutdown(shutdown)
}
