package http

import (
	"net/http"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/config"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/core/ports"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Router struct {
	router *gin.Engine
}

func NewRouter(
	cfg *config.HTTP,
	tokenService ports.TokenService,
	bikeHandler *BikeHandler,
	componentHandler *ComponentHandler,
) (*Router, error) {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.AllowedOrigins},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Swagger
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Metrics
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Bikes routes
	bikes := router.Group("/bikes")
	bikes.Use(AuthMiddleware(tokenService))
	{
		bikes.POST("", bikeHandler.CreateBike)
		bikes.GET("/my", bikeHandler.GetMyBikes)
		bikes.GET("/:id", bikeHandler.GetBike)
		bikes.PUT("/:id", bikeHandler.UpdateBike)
		bikes.DELETE("/:id", bikeHandler.DeleteBike)
		bikes.GET("/:id/with-components", bikeHandler.GetBikeWithComponents)
		bikes.GET("/:id/with-user", bikeHandler.GetBikeWithUser)
	}
	// Components routes
	components := router.Group("/components")
	components.Use(AuthMiddleware(tokenService))
	{
		components.POST("", componentHandler.CreateComponent)
		components.GET("/:id", componentHandler.GetComponent)
		components.PUT("/:id", componentHandler.UpdateComponent)
		components.DELETE("/:id", componentHandler.DeleteComponent)
	}
	return &Router{router: router}, nil
}

func (r *Router) Serve(addr string) error {
	return r.router.Run(addr)
}

func (r *Router) Engine() *gin.Engine {
	return r.router
}
