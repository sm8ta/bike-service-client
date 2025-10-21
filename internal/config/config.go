package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type (
	Container struct {
		App         *App
		Token       *Token
		DB          *DB
		HTTP        *HTTP
		Redis       *Redis
		GRPC        *GRPC
		UserService *UserService
	}

	App struct {
		Name string
		Env  string
	}

	Token struct {
		Secret   string
		Duration string
	}

	DB struct {
		Host     string
		Port     string
		User     string
		Password string
		Name     string
	}

	HTTP struct {
		Env            string
		Port           string
		AllowedOrigins string
		URL            string
	}

	Redis struct {
		Address  string
		Password string
	}

	GRPC struct {
		Port string
	}

	UserService struct {
		Address string
	}
)

func New() (*Container, error) {
	if os.Getenv("APP_ENV") != "production" {
		err := godotenv.Load()
		if err != nil {
			return nil, err
		}
	}

	app := &App{
		Name: os.Getenv("APP_NAME"),
		Env:  os.Getenv("APP_ENV"),
	}

	token := &Token{
		Secret:   os.Getenv("TOKEN_SECRET"),
		Duration: os.Getenv("TOKEN_DURATION"),
	}

	db := &DB{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		Name:     os.Getenv("DB_NAME"),
	}

	http := &HTTP{
		Port:           os.Getenv("HTTP_PORT"),
		AllowedOrigins: os.Getenv("ALLOWED_ORIGINS"),
		URL:            os.Getenv("HTTP_URL"),
		Env:            os.Getenv("APP_ENV"),
	}

	redis := &Redis{
		Address:  os.Getenv("REDIS_ADDRESS"),
		Password: os.Getenv("REDIS_PASSWORD"),
	}

	grpc := &GRPC{
		Port: os.Getenv("GRPC_PORT"),
	}

	userService := &UserService{
		Address: os.Getenv("USER_SERVICE_ADDRESS"),
	}

	return &Container{
		App:         app,
		Token:       token,
		DB:          db,
		HTTP:        http,
		Redis:       redis,
		GRPC:        grpc,
		UserService: userService,
	}, nil
}

func (g *GRPC) PortInt() int {
	port, err := strconv.Atoi(g.Port)
	if err != nil {
		return 50052 // дефолт если ошибка
	}
	return port
}
