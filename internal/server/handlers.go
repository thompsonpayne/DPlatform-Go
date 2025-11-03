// Package server
package server

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"rplatform-echo/cmd/web"
	"rplatform-echo/cmd/web/components/toast"
	"rplatform-echo/internal/repository"

	"github.com/coder/websocket"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/oklog/ulid/v2"
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
		ID:       ulid.Make().String(),
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

// NOTE: create room here
func (s *Server) createRoomHandler(c echo.Context) error {
	name := c.FormValue("name")
	if utf8.RuneCountInString(name) == 0 {
		c.Response().WriteHeader(http.StatusBadRequest)
		if err := toast.Toast(toast.Props{
			Title:       "Room",
			Description: "Can't be empty",
			Variant:     toast.VariantError,
		}).Render(c.Request().Context(), c.Response()); err != nil {
			return err
		}
		return nil
	}

	createdRoom, err := s.roomSvc.Create(c.Request().Context(), name)
	if err != nil {
		c.Response().WriteHeader(http.StatusNotFound)
		if err := toast.Toast(toast.Props{
			Title:       "Room",
			Description: err.Error(),
			Variant:     toast.VariantError,
		}).Render(c.Request().Context(), c.Response()); err != nil {
			return err
		}
		return nil
	}
	return web.Render(c, http.StatusOK, web.RoomCreateResponse(&createdRoom))
}

func (s *Server) deleteRoomHandler(c echo.Context) error {
	id := c.QueryParam("id")
	if err := s.roomSvc.Delete(c.Request().Context(), id); err != nil {
		c.Response().WriteHeader(http.StatusNotFound)
		if err := toast.Toast(toast.Props{
			Title:       "Delete",
			Description: err.Error(),
			Variant:     toast.VariantError,
		}).Render(c.Request().Context(), c.Response()); err != nil {
			return err
		}
		return nil
	}

	return toast.Toast(toast.Props{
		Title:         "Delete",
		Description:   "Success",
		ShowIndicator: true,
		Dismissible:   true,
		Variant:       toast.VariantSuccess,
	}).Render(c.Request().Context(), c.Response())
}

func (s *Server) editRoomHandler(c echo.Context) error {
	id := c.Param("id")
	name := c.FormValue("name")
	err := s.roomSvc.Update(c.Request().Context(), id, name)
	if err != nil {
		return toast.Toast(toast.Props{
			Title:       "Room",
			Description: err.Error(),
			Variant:     toast.VariantError,
		}).Render(c.Request().Context(), c.Response())
	}
	if err := toast.Toast(toast.Props{
		Title:       "Room",
		Description: "Success",
		Variant:     toast.VariantSuccess,
	}).Render(c.Request().Context(), c.Response()); err != nil {
		return err
	}
	return s.getRoomRow(c)
}

func (s *Server) getAllRoomHandler(c echo.Context) error {
	rooms, err := s.roomSvc.List(c.Request().Context())
	if err != nil {
		if err := web.Render(c, http.StatusInternalServerError, web.ErrorMsg(err.Error())); err != nil {
			return err
		}
		return err
	}
	return web.Render(c, http.StatusOK, web.Rooms(rooms))
}

func (s *Server) getChatRoomHanlder(c echo.Context) error {
	id := c.Param("id")
	user := c.Get("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	userID := claims["user_id"].(string)
	email := claims["email"].(string)
	room, err := s.roomSvc.Get(c.Request().Context(), id)
	if err != nil {
		return toast.Toast(toast.Props{
			Title:       "Room",
			Description: err.Error(),
			Variant:     toast.VariantError,
		}).Render(c.Request().Context(), c.Response())
	}
	msgs, err := s.messageSvc.ListFirst(c.Request().Context(), id)
	if err != nil {
		c.Response().WriteHeader(http.StatusInternalServerError)
		return toast.Toast(toast.Props{
			Title:       "Room",
			Description: err.Error(),
			Variant:     toast.VariantError,
		}).Render(c.Request().Context(), c.Response())
	}
	c.Response().Header().Set("HX-Redirect", "/dashboard/"+id)

	if err := web.Render(c, http.StatusOK, web.ChatRoom(&room, userID, email, msgs)); err != nil {
		c.Response().WriteHeader(http.StatusInternalServerError)
		return toast.Toast(toast.Props{
			Title:       "Room",
			Description: err.Error(),
			Variant:     toast.VariantError,
		}).Render(c.Request().Context(), c.Response())
	}
	return nil
}

func (s *Server) getMoreMessagesHandler(c echo.Context) error {
	time.Sleep(time.Millisecond * 300) // NOTE: add delay on purpose, so that scrollbar has time to render to avoid continuous fetching, try removing this line and test scrollbar to see
	roomID := c.Param("roomID")
	createdAt := c.QueryParam("createdAt")

	ca, err := time.Parse(time.RFC3339, createdAt)
	log.Println("Time param query", ca)
	if err != nil {
		log.Println("error converting time", err)
		return err
	}
	msgs, listErr := s.messageSvc.ListNext(c.Request().Context(), roomID, sql.NullTime{Time: ca, Valid: true})
	log.Println("Messages:")
	fmt.Printf("%-5s | %-30s | %-25s\n", "Index", "Content", "CreatedAt")
	fmt.Println(strings.Repeat("-", 70))

	for i, msg := range msgs {
		fmt.Printf("%-5d | %-30s | %-25s\n", i, msg.Content, msg.CreatedAt.Time)
	}
	log.Println("=================")

	if listErr != nil {
		return listErr
	}
	user := c.Get("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	userID := claims["user_id"].(string)
	return web.Render(c, http.StatusOK, web.OlderMessages(msgs, userID))
}

func (s *Server) authHandler(c echo.Context) error {
	cookie, err := c.Cookie("jwt_token")
	if err != nil {
		// cookie not found, render auth page
		return web.Render(c, http.StatusOK, web.Auth())
	}
	if cookie.Value != "" {
		// NOTE: if there's a jwt token in the cookie then redirect page
		if err := c.Redirect(http.StatusFound, "/dashboard"); err != nil {
			return err
		}
	}
	return web.Render(c, http.StatusOK, web.Auth())
}

func (s *Server) getEditRoomForm(c echo.Context) error {
	id := c.Param("id")
	name := c.QueryParam("name")
	return web.Render(c, http.StatusOK, web.RoomEditForm(id, name))
}

func (s *Server) getRoomRow(c echo.Context) error {
	id := c.Param("id")
	room, err := s.roomSvc.Get(c.Request().Context(), id)
	if err != nil {
		return toast.Toast(toast.Props{
			Title:       "Room",
			Description: err.Error(),
			Variant:     toast.VariantError,
		}).Render(c.Request().Context(), c.Response())
	}

	return web.Render(c, http.StatusOK, web.Room(&room))
}
