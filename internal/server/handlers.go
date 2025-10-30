// Package server
package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
	"unicode/utf8"

	"rplatform-echo/cmd/web"
	"rplatform-echo/cmd/web/components/toast"
	"rplatform-echo/internal/repository"

	"github.com/a-h/templ"
	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

func (s *Server) HelloWorldHandler(c echo.Context) error {
	resp := map[string]string{
		"message": "Hello World",
	}

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) healthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, s.db.Health())
}

func (s *Server) websocketHandler(c echo.Context) error {
	w := c.Response().Writer
	r := c.Request()
	socket, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Printf("could not open websocket: %v", err)
		_, _ = w.Write([]byte("could not open websocket"))
		w.WriteHeader(http.StatusInternalServerError)
		return nil
	}

	defer func() {
		if err := socket.Close(websocket.StatusGoingAway, "server closing websocket"); err != nil {
			log.Println("Error closing connection")
		}
	}()

	ctx := r.Context()
	socketCtx := socket.CloseRead(ctx)

	for {
		payload := fmt.Sprintf("server timestamp: %d", time.Now().UnixNano())
		err := socket.Write(socketCtx, websocket.MessageText, []byte(payload))
		if err != nil {
			break
		}
		time.Sleep(time.Second * 2)
	}
	return nil
}

func (s *Server) registerHanlder(c echo.Context) error {
	password := c.FormValue("password")
	email := c.FormValue("email")
	errorMsg := ""

	// Basic validation (add more robust validation as needed)
	if email == "" || password == "" {
		errorMsg = "Email and password are required"
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		errorMsg = "Could not hash password"
	}

	// Create a new User object
	user := repository.CreateUserParams{
		ID:       uuid.New().String(),
		Name:     email,
		Email:    email,
		Password: string(hashedPassword),
	}

	rawDB := s.db.GetDB()
	queries := repository.New(rawDB)

	// Save the user to the database
	_, error := queries.CreateUser(c.Request().Context(), user)
	if error != nil {
		errorMsg = error.Error()
	}
	component := web.UserResponse(user.Name, user.Email, errorMsg)

	return web.Render(c, http.StatusOK, component)
}

func (s *Server) loginHandler(c echo.Context) error {
	password := c.FormValue("password")
	email := c.FormValue("email")
	errorMsg := ""

	// Basic validation (add more robust validation as needed)
	if email == "" || password == "" {
		errorMsg = "Email and password are required"
	}

	rawDB := s.db.GetDB()
	// Get the user from db to get password for reference
	queries := repository.New(rawDB)
	userResult, err := queries.GetUserByEmail(c.Request().Context(), email)
	if err != nil {
		errorMsg = "Internal Error"
	}

	// Check the password
	if !web.CheckPasswordHash(password, userResult.Password) {
		errorMsg = "Invalid credentials"
	}

	token, err := web.GenerateToken(userResult.ID, userResult.Email)
	if err != nil {
		errorMsg = "Internal error"
	}

	cookie := &http.Cookie{
		Name:     "jwt_token",
		Value:    token,
		MaxAge:   86400, // 24 hours
		HttpOnly: true,
		Path:     "/",
		Secure:   false, // HTTPS only
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(cookie)
	// Use HX-Redirect for full page navigation instead of inline content
	if errorMsg == "" {
		c.Response().Header().Set("HX-Redirect", "/dashboard")
		return c.String(http.StatusOK, "OK")
	}

	component := web.UserResponse(email, email, errorMsg)
	return web.Render(c, http.StatusOK, component)
}

func (s *Server) logOutHandler(c echo.Context) error {
	cookie := &http.Cookie{
		Name:     "jwt_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(cookie)
	c.Response().Header().Set("HX-Redirect", "/auth")
	return c.String(http.StatusFound, "OK")
}

func (s *Server) createRoom(c echo.Context) (repository.Room, error) {
	name := c.FormValue("name")
	if utf8.RuneCountInString(name) == 0 {
		return repository.Room{}, errors.New("can't be empty")
	}
	room := repository.CreateRoomParams{
		ID:   uuid.New().String(),
		Name: name,
	}

	rawDB := s.db.GetDB()
	queries := repository.New(rawDB)

	// Save the user to the database
	createdRoom, err := queries.CreateRoom(c.Request().Context(), room)
	return createdRoom, err
}

func (s *Server) deleteRoom(c echo.Context, id string) error {
	rawDB := s.db.GetDB()
	queries := repository.New(rawDB)

	// Save the user to the database
	err := queries.DeleteRoom(c.Request().Context(), id)
	if err != nil {
		c.Response().WriteHeader(http.StatusNotFound)
		toast.Toast(toast.Props{
			Title:       "Delete",
			Description: err.Error(),
			Variant:     toast.VariantError,
			Attributes:  templ.Attributes{"hx-swap-oob": "true"},
		}).Render(c.Request().Context(), c.Response())
		return nil
	}

	toast.Toast(toast.Props{
		Title:         "Delete",
		Description:   "Success",
		ShowIndicator: true,
		Dismissible:   true,
		Variant:       toast.VariantSuccess,
	}).Render(c.Request().Context(), c.Response())
	return nil
}

func (s *Server) getAllRoom(c echo.Context) ([]repository.Room, error) {
	rawDB := s.db.GetDB()
	queries := repository.New(rawDB)
	rooms, err := queries.GetAllRooms(c.Request().Context())
	return rooms, err
}
