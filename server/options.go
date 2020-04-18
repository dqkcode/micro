package server

import (
	"fmt"
	"net/http"
	"net/textproto"
	"os"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pthethanh/micro/auth"
	"github.com/pthethanh/micro/auth/jwt"
	"github.com/pthethanh/micro/health"
	"github.com/pthethanh/micro/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	defaultAddr = ":8000"
)

// StreamInterceptors is an option allows add additional stream interceptors to the server.
func StreamInterceptors(interceptors ...grpc.StreamServerInterceptor) Option {
	return func(opts *Server) {
		opts.streamInterceptors = append(opts.streamInterceptors, interceptors...)
	}
}

// UnaryInterceptors is an option allows add additional unary interceptors to the server.
func UnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) Option {
	return func(opts *Server) {
		opts.unaryInterceptors = append(opts.unaryInterceptors, interceptors...)
	}
}

// JWTAuth is an option allow to add jwt authenticator to the server.
func JWTAuth(secret string) Option {
	return func(opts *Server) {
		if secret == "" {
			return
		}
		f := jwt.Authenticator([]byte(secret))
		opts.streamInterceptors = append(opts.streamInterceptors, auth.StreamInterceptor(f))
		opts.unaryInterceptors = append(opts.unaryInterceptors, auth.UnaryInterceptor(f))
	}
}

// WithAuth is an option allow to add an authenticator to the server.
func WithAuth(f auth.AuthenticatorFunc) Option {
	return func(opts *Server) {
		opts.streamInterceptors = append(opts.streamInterceptors, auth.StreamInterceptor(f))
		opts.unaryInterceptors = append(opts.unaryInterceptors, auth.UnaryInterceptor(f))
	}
}

// Logger is an option allow add a custom logger into the server.
func Logger(logger log.Logger) Option {
	return func(opts *Server) {
		opts.log = logger
		opts.serveMuxOptions = append(opts.serveMuxOptions, DefaultHeaderMatcher())
		opts.streamInterceptors = append(opts.streamInterceptors, log.StreamInterceptor(logger))
		opts.unaryInterceptors = append(opts.unaryInterceptors, log.UnaryInterceptor(logger))
	}
}

// TLS is an option allow add TLS for transport security to the server.
func TLS(key, cert string) Option {
	return func(opts *Server) {
		if key == "" || cert == "" {
			return
		}
		opts.tlsKeyFile = key
		opts.tlsCertFile = cert
		creds, err := credentials.NewServerTLSFromFile(opts.tlsCertFile, opts.tlsKeyFile)
		if err != nil {
			panic(err)
		}
		opts.serverOptions = append(opts.serverOptions, grpc.Creds(creds))
	}
}

// MetricsPaths is an option allow override readiness, liveness and metrics path.
func MetricsPaths(ready, live, metrics string) Option {
	return func(opts *Server) {
		opts.readinessPath = ready
		opts.livenessPath = live
		opts.metricsPath = metrics
	}
}

// Timeout is an option to override default read/write timeout.
func Timeout(read, write time.Duration) Option {
	if read == 0 {
		read = 30 * time.Second
	}
	if write == 0 {
		write = 30 * time.Second
	}
	return func(opts *Server) {
		opts.readTimeout = read
		opts.writeTimeout = write
		opts.serverOptions = append(opts.serverOptions, grpc.ConnectionTimeout(read))
	}
}

// ServeMuxOptions is an option allow add additional ServeMuxOption.
func ServeMuxOptions(muxOpts ...runtime.ServeMuxOption) Option {
	return func(opts *Server) {
		opts.serveMuxOptions = append(opts.serveMuxOptions, muxOpts...)
	}
}

// Options is an option allow add additional grpc.ServerOption.
func Options(serverOpts ...grpc.ServerOption) Option {
	return func(opts *Server) {
		opts.serverOptions = append(opts.serverOptions, serverOpts...)
	}
}

// HealthChecks is an option allow set health check function.
func HealthChecks(checks ...health.CheckFunc) Option {
	return func(opts *Server) {
		opts.healthChecks = append(opts.healthChecks, checks...)
	}
}

// AddressFromEnv is an option to get address from environment configuration.
// It looks for PORT and then ADDRESS variables.
// This option is mostly used for cloud environment like Heroku where the port
// is randomly set.
func AddressFromEnv() Option {
	return func(opts *Server) {
		if p := os.Getenv("PORT"); p != "" {
			opts.address = fmt.Sprintf(":%s", p)
			return
		}
		if addr := os.Getenv("ADDRESS"); addr != "" {
			opts.address = addr
			return
		}
		if opts.address == "" {
			opts.address = defaultAddr
		}
	}
}

// HTTPOnly is an option to set additional HTTP only handler.
// This is mostly used for adding additional internal API or serve static files.
// *NOTE: This method will panic if path / is provided.
func HTTPOnly(path string, method string, h http.Handler, queries ...string) Option {
	return func(opts *Server) {
		if path == "/" {
			log.Panic("Using path / will cause issue with gRPC routing and hence is not allowed.")
		}
		opts.getOrCreateRouter().Path(path).Methods(method).Queries(queries...).Handler(h)
	}
}

// DefaultHeaderMatcher return a ServerMuxOption that forward
// header keys request-id, api-key to GRPC Context.
func DefaultHeaderMatcher() runtime.ServeMuxOption {
	return HeaderMatcher([]string{"Request-Id", "Api-Key"})
}

// HeaderMatcher return a serveMuxOption for matcher header
// for passing a set of non IANA headers to GRPC context
// without a need to prefix them with Grpc-Metadata.
func HeaderMatcher(keys []string) runtime.ServeMuxOption {
	return runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
		canonicalKey := textproto.CanonicalMIMEHeaderKey(key)
		for _, k := range keys {
			if k == canonicalKey || textproto.CanonicalMIMEHeaderKey(k) == canonicalKey {
				return k, true
			}
		}
		return runtime.DefaultHeaderMatcher(key)
	})
}
