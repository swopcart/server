package www

import "github.com/swopcart/server/www/handlers"

func (srv *Server) routes() {
	srv.router.GET("/", handlers.Root)
	srv.router.GET("/ping", handlers.Ping)

	apiRouter := srv.router.Group("/api")
	{
		_ = apiRouter
	}

	adminRouter := srv.router.Group("/admin")
	{
		_ = adminRouter
	}
}
