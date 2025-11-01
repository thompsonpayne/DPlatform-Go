package server

import (
	"net/http"
	"os"

	"rplatform-echo/cmd/web"
	"rplatform-echo/internal/ws"

	"github.com/a-h/templ"

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

	e.GET("/auth", s.authHandler)
	{
		auth := e.Group("/auth")
		auth.POST("/login", s.loginHandler)
		auth.POST("/register", s.registerHanlder)
		auth.POST("/logout", s.logOutHandler)
	}

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
			return web.Render(c, http.StatusOK, web.DashBoard())
		})

		d.GET("/room-edit/:id", s.getEditRoomForm)
		d.GET("/room-row/:id", s.getRoomRow)

		roomManager := ws.NewRoomManager()

		// NOTE: Room chat UI
		d.GET("/:id", func(c echo.Context) error {
			roomID := c.Param("id")
			roomManager.GetRoom(roomID)

			return s.getChatRoomHanlder(c)
		})

		d.GET("/api/room", s.getAllRoomHandler)

		d.POST("/api/room", s.createRoomHandler)

		d.PATCH("/api/room/:id", s.editRoomHandler)

		d.DELETE("/api/room", s.deleteRoomHandler)

		d.GET("/chatroom/:id", func(c echo.Context) error {
			roomID := c.Param("id")
			room := roomManager.GetRoom(roomID)
			return ws.ServeWs(room, c)
		})
	}

	e.GET("/", s.HelloWorldHandler)

	e.GET("/health", s.healthHandler)

	e.GET("/websocket", s.websocketHandler)

	return e
}
