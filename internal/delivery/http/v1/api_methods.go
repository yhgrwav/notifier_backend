package v1

import (
	"RedCollar/internal/domain"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *Handler) checkLocation(c *gin.Context) {
	//Создаем переменную в которую будем записывать ответ
	var request domain.LocationCheckRequest

	//Анмаршалим реквест в переменную
	err := c.ShouldBindJSON(&request)
	if err != nil {
		c.JSON(400, gin.H{ //до вызова сервиса мы обрабатываем кейс когда ошибка возникает по вине пользователя
			"Ошибка": "Невалидное тело запроса",
		})
		return
	}

	//DefaultQuery проверяет не пришёл ли нам query параметр
	//если пришел - записываем в переменную
	//если не пришел - ставим дефолт
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	//Когда у нас готовы все аргументы - вызываем метод сервиса
	resp, err := h.service.CheckLocation(c.Request.Context(), request, limit, offset)
	if err != nil {
		c.JSON(500, gin.H{ //после вызова сервиса когда мы уверены, что полученные данные валидные
			"Ошибка": err.Error(), // отдаём на потенциальную ошибку статус код 500 и распаковываем ошибку

		})
		return
	}

	//Если нет ошибок и всё ок - отдаём ок и тело ответа
	c.JSON(200, resp)
}

// GetStats реализует требования тз, отдавая статистику по зонам  по запросу GET /api/v1/incidents/stats
func (h *Handler) GetStats(c *gin.Context) {
	//читать какие-то данные от пользователя нам не нужно, так что просто сразу же вызываем сервис
	result, err := h.service.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{"Ошибка": err.Error()}) //обрабатываем единственный кейс когда у нас может что-то поломаться
		return
	}
	c.JSON(200, result) //и возвращаем полученный результат
}
