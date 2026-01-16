package www

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/swopcart/server/frontend"
)

func (srv *Server) routes() {
	srv.apiRoutes()
	srv.webRoutes()
}

func (srv *Server) apiRoutes() {
	api := srv.Router.Group("/api")

	v0 := api.Group("/v0")
	srv.v0.InstallRoutes(v0)
}

func (srv *Server) webRoutes() {
	distFS, _ := fs.Sub(frontend.Content, "dist")
	srv.Router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.Status(http.StatusNotFound)
			return
		}

		f, err := distFS.Open(c.Request.URL.Path[1:]) // strip leading /
		if err == nil {
			_ = f.Close()
			http.FileServer(http.FS(distFS)).ServeHTTP(c.Writer, c.Request)
			return
		}

		c.FileFromFS("/", http.FS(distFS))
	})
}
