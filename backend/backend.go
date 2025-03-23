package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

const flaskServer = "http://localhost:5000/predict"

func main() {
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "upload.html", nil)
	})

	r.POST("/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.String(http.StatusBadRequest, "Ошибка загрузки файла")
			return
		}

		filePath := filepath.Join("uploads", file.Filename)
		if err := c.SaveUploadedFile(file, filePath); err != nil {
			c.String(http.StatusInternalServerError, "Ошибка сохранения файла")
			return
		}

		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/predict?file=%s", filePath))
	})

	r.GET("/predict", func(c *gin.Context) {
		filePath := c.Query("file")
		if filePath == "" {
			c.String(http.StatusBadRequest, "Файл не указан")
			return
		}

		file, err := os.Open(filePath)
		if err != nil {
			c.String(http.StatusInternalServerError, "Ошибка открытия файла")
			return
		}
		defer file.Close()

		var requestBody bytes.Buffer
		writer := multipart.NewWriter(&requestBody)

		part, err := writer.CreateFormFile("file", filepath.Base(filePath))
		if err != nil {
			c.String(http.StatusInternalServerError, "Ошибка формирования запроса")
			return
		}

		_, err = io.Copy(part, file)
		if err != nil {
			c.String(http.StatusInternalServerError, "Ошибка копирования файла")
			return
		}

		writer.Close()

		resp, err := http.Post(flaskServer, writer.FormDataContentType(), &requestBody)
		if err != nil {
			c.String(http.StatusInternalServerError, "Ошибка отправки запроса")
			return
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			c.String(http.StatusInternalServerError, "Ошибка обработки ответа")
			return
		}

		predictions, _ := result["prediction"].([]interface{})
		dates, _ := result["dates"].([]any) // ого штуку нашел это аллиас на интерфейс))))
		imageBase64, _ := result["image_base64"].(string)

		changeText := generateRecommendation(predictions)

		c.HTML(http.StatusOK, "result.html", gin.H{
			"predictions":  predictions,
			"dates":        dates,
			"change":       changeText,
			"image_base64": imageBase64,
		})
	})

	r.Run(":8082")
}
func generateRecommendation(predictionsRaw []interface{}) string {

	predictions := make([]float64, len(predictionsRaw))
	for i, v := range predictionsRaw {
		predictions[i] = v.(float64)
	}

	if len(predictions) < 2 {
		return "Недостаточно данных для прогноза."
	}
	// Считаем разницу между последним прогнозом и текущей ценой
	currentPrice := predictions[0]
	nextWeekPrice := predictions[1]

	if nextWeekPrice > currentPrice {
		return "Прогнозируется рост цен. Рекомендуется сделать более масштабную закупку (например, 2X-5X)."
	} else if nextWeekPrice < currentPrice {
		return "Прогнозируется снижение цен. Закупайтесь на 1 неделю (X тонн), не стоит увеличивать объем."
	} else {
		return "Цена останется на том же уровне. Можно закупаться в стандартном объеме (X тонн)."
	}
}
