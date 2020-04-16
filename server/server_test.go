package server_test

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/pthethanh/micro/log"
	"github.com/pthethanh/micro/server"
	"google.golang.org/grpc"
)

func TestInitServerDefault(t *testing.T) {
	addr := ":8000"
	os.Setenv("ADDRESS", addr)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Warn("address is already in use, ignore unit test")
		t.SkipNow()
		return
	}
	l.Close() // close to start the test

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := server.ListenAndServeContext(ctx); err != nil {
		if err != ctx.Err() {
			t.Error(err)
		}
	}
}

func TestInitServerWithOptions(t *testing.T) {
	addr := ":8001"
	os.Setenv("ADDRESS", addr)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Warn("address is already in use, ignore unit test")
		t.SkipNow()
		return
	}
	l.Close() // close to start the test

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	srv := server.New("",
		server.AddressFromEnv(),
		server.JWTAuth("secret"),
		server.Logger(log.Root()),
		server.MetricsPaths("ready", "live", "metrics"),
		server.ServeMuxOptions(server.DefaultHeaderMatcher()),
		server.Options(grpc.ConnectionTimeout(20*time.Second)),
		server.Timeout(20*time.Second, 20*time.Second),
		server.HealthChecks(func(ctx context.Context) error {
			return nil
		}),
	)
	if err := srv.ListenAndServeContext(ctx); err != nil {
		if err != ctx.Err() {
			t.Error(err)
		}
	}
}
