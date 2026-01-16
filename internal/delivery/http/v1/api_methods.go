package v1

import (
	"RedCollar/internal/domain"
	"strconv"

	"github.com/gin-gonic/gin"
)

// POST /api/v1/location/check
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
	result, err := h.service.GetStats(c.Request.Context(), h.statsTime)
	if err != nil {
		c.JSON(500, gin.H{"Ошибка": err.Error()}) //обрабатываем единственный кейс когда у нас может что-то поломаться
		return
	}
	c.JSON(200, result) //и возвращаем полученный результат
}

// POST /api/v1/incidents
func (h *Handler) CreateIncident(c *gin.Context) {
	//создаем переменную в которую записываем полученные параметры инцидента
	var input domain.Incident
	if err := c.ShouldBindJSON(&input); err != nil { //если ошибка - отдаём ошибку
		c.JSON(400, gin.H{"Ошибка": err.Error()})
		return
	}

	//вызываем сервис с переданной структурой
	id, err := h.service.Create(c.Request.Context(), &input)
	if err != nil {
		c.JSON(500, gin.H{"Ошибка": err.Error()}) //если ошибка - ошибка
		return
	}
	c.JSON(200, gin.H{"id": id}) // если всё ок - отдаём ок и айди созданного инцидента
}

// GET /api/v1/incidents/:id
func (h *Handler) GetIncidentByID(c *gin.Context) {
	// берём id из url
	id := c.Param("id")

	//вызываем сервис с айди в аргументах
	result, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil { //если ошибка - отдаём ошибку
		c.JSON(500, gin.H{"Ошибка": err.Error()})
		return
	}

	//если всё ок - отдаём ок и результат
	c.JSON(200, result)
}

// DELETE /api/v1/incidents/:id (деактивация)
func (h *Handler) DeleteIncident(c *gin.Context) {
	//читаем id из url
	id := c.Param("id")

	//вызываем сервис с переданным id
	err := h.service.Delete(c.Request.Context(), id)
	if err != nil {
		c.JSON(500, gin.H{"Ошибка": err.Error()}) // если ошибка - отдаём ошибку
		return
	}

	//если всё ок - отдаём ок и id деактивированного инцидента
	c.JSON(200, gin.H{"id": id})
}

// PUT /api/v1/incidents/:id
func (h *Handler) UpdateIncident(c *gin.Context) {
	//поулчаем id из url
	id := c.Param("id")

	//создаем структуру инцидента в которую записываем новые данные
	var input domain.Incident
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"Ошибка": err.Error()}) //если ошибка - отдаём ошибку
		return
	}
	uuid, err := h.service.Update(c.Request.Context(), id, &input)
	if err != nil {
		c.JSON(500, gin.H{"Ошибка": err.Error()})
		return
	}
	c.JSON(200, gin.H{"UUID": uuid})
}

// GET /api/v1/incidents
func (h *Handler) GetIncidents(c *gin.Context) {
	//получаем координаты, параметры пагинации
	lat, _ := strconv.ParseFloat(c.DefaultQuery("latitude", "0"), 64)
	lng, _ := strconv.ParseFloat(c.DefaultQuery("longitude", "0"), 64)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	//передаем всё в аргументы метода сервиса
	result, err := h.service.Get(c.Request.Context(), lat, lng, limit, offset)
	if err != nil {
		c.JSON(500, gin.H{"Ошибка": err.Error()}) // если ошибка - отдаём ошибку
		return
	}
	c.JSON(200, result) //если всё ок - отдаём ок и результат
}

// GET /api/v1/system/health
func (h *Handler) GetHealth(c *gin.Context) {
	c.JSON(200, gin.H{"всё": "ок"})
}
