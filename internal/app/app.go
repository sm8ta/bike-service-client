package app

import (
	"context"
	"database/sql"
	"fmt"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/adapter/handler/http"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/adapter/logger"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/adapter/postgres"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/adapter/prometheus"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/adapter/redis"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/config"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/core/ports"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/core/services"
	user_client "github.com/sm8ta/webike_user_microservice_nikita/pkg/client"

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
		return nil, fmt.Errorf("Failed to connect to database:%w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("Failed to ping database:%w", err)
	}

	// Migrate DB
	if err := goose.Up(db, "./internal/adapter/postgres/migrations"); err != nil {
		return nil, fmt.Errorf("Failed to run migrations:%w", err)
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

	// User service client init
	transport := httptransport.New("localhost:8080", "", []string{"http"})
	userClient := user_client.New(transport, strfmt.Default)

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
	}, nil
}

// Runs all services
func (a *App) Run() error {
	listenAddr := fmt.Sprintf("%s:%s", a.Config.HTTP.URL, a.Config.HTTP.Port)
	a.Logger.Info("Starting HTTP server", map[string]interface{}{
		"addr": listenAddr,
	})

	if err := a.HTTPRouter.Serve(listenAddr); err != nil {
		a.Logger.Error("HTTP server error", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}
	return nil
}

// Stops all services
func (a *App) Stop(ctx context.Context) error {
	a.Logger.Info("Shutting down gracefully...", nil)

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
