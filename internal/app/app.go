package app

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"webike_bike_microservice_nikita/internal/adapter/handler/http"
	"webike_bike_microservice_nikita/internal/adapter/logger"
	"webike_bike_microservice_nikita/internal/adapter/postgres"
	"webike_bike_microservice_nikita/internal/adapter/prometheus"
	"webike_bike_microservice_nikita/internal/adapter/redis"
	grpcapp "webike_bike_microservice_nikita/internal/app/grpc"
	"webike_bike_microservice_nikita/internal/config"
	"webike_bike_microservice_nikita/internal/core/ports"
	"webike_bike_microservice_nikita/internal/core/services"
	"webike_bike_microservice_nikita/internal/grpc"

	"github.com/go-playground/validator/v10"
	"github.com/pressly/goose"
	redisClient "github.com/redis/go-redis/v9"
)

type App struct {
	Config       *config.Container
	Logger       ports.LoggerPort
	DB           *sql.DB
	RedisClient  *redisClient.Client
	RedisAdapter ports.CachePort
	HTTPRouter   *http.Router
	GRPCServer   *grpcapp.App
	UserClient   *grpc.UserClient
}

func New(ctx context.Context, cfg *config.Container) (*App, error) {
	// Set logger
	loggerAdapter := logger.NewLoggerAdapter(cfg.App.Env)
	loggerAdapter.Info("Starting the application", map[string]interface{}{
		"app": cfg.App.Name,
		"env": cfg.App.Env,
	})

	// Set redis
	redisConn := redisClient.NewClient(&redisClient.Options{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       0,
	})
	if _, err := redisConn.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	cacheAdapter := redis.NewRedisAdapter(redisConn)

	// Connect DB
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DB.Host, cfg.DB.Port, cfg.DB.User, cfg.DB.Password, cfg.DB.Name)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to database: ", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("Failed to ping database:%w", err)
	}

	// Migrate DB
	if err := goose.Up(db, "./internal/adapter/postgres/migrations"); err != nil {
		return nil, fmt.Errorf("Failed to run migrations: ", err)
	}
	// Validate
	validate := validator.New()

	// Observability
	metrics := prometheus.NewPrometheusAdapter()

	// Repositories
	bikeRepo := postgres.NewBikeRepository(db)
	componentRepo := postgres.NewComponentRepository(db)

	// Services
	bikeService := services.NewBikeService(bikeRepo, componentRepo, loggerAdapter, validate, cacheAdapter)
	componentService := services.NewComponentService(componentRepo, loggerAdapter, validate, cacheAdapter)

	// gRPC Server
	grpcServer := grpcapp.New(loggerAdapter, bikeService, cfg.GRPC.PortInt())

	// gRPC User Client
	userClient, err := grpc.NewUserClient(
		ctx,
		loggerAdapter,
		cfg.UserService.Address,
		5*time.Second,
		3,
	)
	if err != nil {
		loggerAdapter.Warn("Failed to connect to User Service", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// HTTP Handlers
	tokenService := http.NewJWTTokenService(cfg.Token.Secret, loggerAdapter)
	bikeHandler := http.NewBikeHandler(bikeService, loggerAdapter, metrics, userClient)
	componentHandler := http.NewComponentHandler(componentService, bikeService, loggerAdapter, metrics)

	// Init HTTP router
	router, err := http.NewRouter(
		cfg.HTTP,
		tokenService,
		bikeHandler,
		componentHandler,
	)
	if err != nil {
		db.Close()
		redisConn.Close()
		return nil, fmt.Errorf("failed to initialize router: %w", err)
	}

	return &App{
		Config:       cfg,
		Logger:       loggerAdapter,
		DB:           db,
		RedisClient:  redisConn,
		RedisAdapter: cacheAdapter,
		HTTPRouter:   router,
		GRPCServer:   grpcServer,
		UserClient:   userClient,
	}, nil
}

// Runs all services
func (a *App) Run() {

	// Start gRPC server
	go a.GRPCServer.MustRun()

	// Start HTTP server
	go func() {
		listenAddr := fmt.Sprintf("%s:%s", a.Config.HTTP.URL, a.Config.HTTP.Port)
		a.Logger.Info("Starting HTTP server", map[string]interface{}{
			"addr": listenAddr,
		})

		if err := a.HTTPRouter.Serve(listenAddr); err != nil {
			a.Logger.Error("HTTP server error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	a.Logger.Info("Application is running", nil)
}

// Stops all services
func (a *App) Stop(ctx context.Context) error {
	a.Logger.Info("Shutting down gracefully...", nil)

	// Stop gRPC server
	a.GRPCServer.Stop()

	// Close database
	if err := a.DB.Close(); err != nil {
		a.Logger.Error("Database close error", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Close Redis
	if err := a.RedisClient.Close(); err != nil {
		a.Logger.Error("Redis close error", map[string]interface{}{
			"error": err.Error(),
		})
	}

	a.Logger.Info("Application stopped successfully", nil)
	return nil
}
