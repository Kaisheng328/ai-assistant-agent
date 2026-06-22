package route

import (
	"net/http"
	"strings"

	"api_go/http/controller/chat"
	"api_go/http/controller/knowledge"
	"api_go/http/controller/ollama"
	"api_go/http/controller/setting"

	"goyave.dev/goyave/v4"
	"goyave.dev/goyave/v4/cors"
)

func Register(router *goyave.Router) {
	router.CORS(cors.Default())

	api := router.Subrouter("/api")

	api.Get("/conversations", chat.Index)
	api.Post("/conversations", chat.Create)
	api.Delete("/conversations/{id}", chat.Delete)

	api.Get("/conversations/{id}/messages", chat.Messages)
	api.Post("/conversations/{id}/messages", chat.SendMessage)

	api.Get("/settings", setting.Index)
	api.Post("/settings", setting.Update)

	api.Get("/ollama/models", ollama.Models)

	api.Get("/knowledge", knowledge.Index)
	api.Post("/knowledge", knowledge.Upload)
	api.Delete("/knowledge", knowledge.DeleteAll)
	api.Delete("/knowledge/{id}", knowledge.Delete)

	router.Static("/", "/frontend/dist", false)

	router.StatusHandler(func(response *goyave.Response, request *goyave.Request) {
		if !strings.HasPrefix(request.Request().URL.Path, "/api") {
			response.File("/frontend/dist/index.html")
			return
		}
		response.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}, http.StatusNotFound)
}
