package setting

import (
	"encoding/json"
	"api_go/database/model"
	"goyave.dev/goyave/v4"
	"goyave.dev/goyave/v4/database"
)

func Index(response *goyave.Response, request *goyave.Request) {
	var settings []model.Setting
	if err := database.Conn().Find(&settings).Error; err != nil {
		response.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	
	result := make(map[string]string)
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	response.JSON(200, result)
}

func Update(response *goyave.Response, request *goyave.Request) {
	var body map[string]string
	if err := json.NewDecoder(request.Request().Body).Decode(&body); err != nil {
		response.JSON(400, map[string]string{"error": "Invalid request payload"})
		return
	}

	for k, v := range body {
		setting := model.Setting{Key: k}
		database.Conn().FirstOrCreate(&setting, model.Setting{Key: k})
		setting.Value = v
		database.Conn().Save(&setting)
	}

	response.JSON(200, map[string]string{"status": "settings updated"})
}
