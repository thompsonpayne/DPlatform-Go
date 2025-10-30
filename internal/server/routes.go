package server

import (
	"context"
	"log"
	"net/http"
	"os"

	"rplatform-echo/cmd/web"
	"rplatform-echo/internal/ws"

	"github.com/a-h/templ"
	"github.com/golang-jwt/jwt/v5"

	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func (s *Server) RegisterRoutes() http.Handler {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"https://*", "http://*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	fileServer := http.FileServer(http.FS(web.Files))
	e.GET("/assets/*", echo.WrapHandler(fileServer))

	e.GET("/web", echo.WrapHandler(templ.Handler(web.HelloForm())))
	e.POST("/hello", echo.WrapHandler(http.HandlerFunc(web.HelloWebHandler)))

	e.GET("/auth", echo.WrapHandler(templ.Handler(web.Auth())))
	auth := e.Group("/auth")
	auth.POST("/login", s.loginHandler)
	auth.POST("/register", s.registerHanlder)
	auth.POST("/logout", s.logOutHandler)

	d := e.Group("/dashboard")
	{

		d.Use(echojwt.WithConfig(echojwt.Config{
			TokenLookup: "header:Authorization:Bearer ,cookie:jwt_token",
			SigningKey:  []byte(os.Getenv("JWT_SECRET")),
			ErrorHandler: func(c echo.Context, err error) error {
				return c.Redirect(http.StatusFound, "/auth")
			},
		}))

		// d.GET("", echo.WrapHandler(templ.Handler(web.DashBoard())))
		// d.GET("", echo.WrapHandler(templ.Handler(web.DashBoard())))
		d.GET("", func(c echo.Context) error {
			templ.Handler(web.DashBoard()).ServeHTTP(c.Response(), c.Request())
			return nil
		})

		d.GET("/api/room", func(c echo.Context) error {
			rooms, err := s.getAllRoom(c)
			if err != nil {
				if err := web.Render(c, http.StatusInternalServerError, web.ErrorMsg(err.Error())); err != nil {
					return err
				}
				return err
				// templ.Handler(web.ErrorMsg(err.Error())).ServeHTTP(c.Response(), c.Request())
			}
			// templ.Handler(web.Rooms(rooms)).ServeHTTP(c.Response(), c.Request())
			err = web.Render(c, http.StatusOK, web.Rooms(rooms))
			return err
		})

		d.POST("/api/room", func(c echo.Context) error {
			room, err := s.createRoom(c)
			if err != nil {
				c.Response().Header().Set("HX-Retarget", "#create-room-error-container")
				// templ.Handler(web.ErrorMsg(err.Error())).ServeHTTP(c.Response(), c.Request())
				if err := web.Render(c, http.StatusBadRequest, web.ErrorMsg(err.Error())); err != nil {
					log.Println("Error create room", err)
				}
				return nil
			}
			// templ.Handler(web.Room(&room)).ServeHTTP(c.Response(), c.Request())
			return web.Render(c, http.StatusOK, web.Room(&room))
		})

		d.DELETE("/api/room", func(c echo.Context) error {
			id := c.QueryParam("id")
			return s.deleteRoom(c, id)
		})

		hub := ws.NewHub()
		go hub.Run(context.Background())
		d.GET("/chatroom", func(c echo.Context) error {
			user := c.Get("user").(*jwt.Token)
			claims := user.Claims.(jwt.MapClaims)
			return ws.ServeWs(hub, c.Response(), c.Request(), claims)
		})
	}

	e.GET("/", s.HelloWorldHandler)

	e.GET("/health", s.healthHandler)

	e.GET("/websocket", s.websocketHandler)

	return e
}
