package grpcapp

import (
	"fmt"
	"net"

	"webike_bike_microservice_nikita/internal/core/ports"
	"webike_bike_microservice_nikita/internal/core/services"
	grpcHandler "webike_bike_microservice_nikita/internal/grpc"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type App struct {
	log        ports.LoggerPort
	gRPCServer *grpc.Server
	port       int
}

// New gRPC server app.
func New(
	log ports.LoggerPort,
	bikeService *services.BikeService,
	port int,
) *App {
	loggingOpts := []logging.Option{
		logging.WithLogOnEvents(
			logging.PayloadReceived, logging.PayloadSent,
		),
	}

	// Recovery after panic
	recoveryOpts := []recovery.Option{
		recovery.WithRecoveryHandler(func(p interface{}) (err error) {
			log.Error("Recovered from panic in gRPC handler", map[string]interface{}{
				"panic": p,
			})
			return status.Errorf(codes.Internal, "internal error")
		}),
	}

	// Creates gRPC server with Interceptor
	gRPCServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			recovery.UnaryServerInterceptor(recoveryOpts...),
			logging.UnaryServerInterceptor(grpcHandler.InterceptorLogger(log), loggingOpts...),
		),
	)

	// Registers BikeService
	grpcHandler.Register(gRPCServer, bikeService, log)

	return &App{
		log:        log,
		gRPCServer: gRPCServer,
		port:       port,
	}
}

// MustRun runs gRPC server and panics if any error occurs.
func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

// Run runs gRPC server.
func (a *App) Run() error {
	const op = "grpcapp.Run"

	//TCP listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	a.log.Info("Starting gRPC server", map[string]interface{}{
		"addr": listener.Addr().String(),
	})

	// Starts gRPC server
	if err := a.gRPCServer.Serve(listener); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// Stop stops gRPC server.
func (a *App) Stop() {
	const op = "grpcapp.Stop"

	a.log.Info("Stopping gRPC server", map[string]interface{}{
		"op":   op,
		"port": a.port,
	})

	a.gRPCServer.GracefulStop()
}
