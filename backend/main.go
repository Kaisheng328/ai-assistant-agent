package main

import (
	"os"

	"api_go/database/model"
	"api_go/http/route"

	"goyave.dev/goyave/v4"
	"goyave.dev/goyave/v4/database"
	_ "goyave.dev/goyave/v4/database/dialect/postgres"
)

func main() {
	goyave.RegisterStartupHook(func() {
		database.Conn().AutoMigrate(&model.Conversation{}, &model.Message{}, &model.Setting{})
	})

	if err := goyave.Start(route.Register); err != nil {
		os.Exit(1)
	}
}
