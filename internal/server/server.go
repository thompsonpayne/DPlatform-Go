package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"rplatform-echo/internal/database"
	"rplatform-echo/internal/repository"
	"rplatform-echo/internal/services"
)

type Server struct {
	port int

	db         database.Service
	roomSvc    *services.RoomService
	messageSvc *services.MessageService
}

func NewServer() *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))

	db := database.New()
	// Initialize database schema
	if err := db.InitSchema(); err != nil {
		log.Fatalf("Failed to initialize database schema: %v", err)
	}

	// Wire repository and services
	repo := repository.New(db.GetDB())
	roomSvc := services.NewRoomService(repo)
	messageSvc := services.NewMessageService(repo)

	NewServer := &Server{
		port:       port,
		db:         db,
		roomSvc:    roomSvc,
		messageSvc: messageSvc,
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
