package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MiddlewareAuth отвечает за то, чтобы проверить валидность ключа в заголовке для выдачи доступа к роли оператора
func MiddlewareAuth(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		//получаем ключ из заголовка
		headerKey := c.GetHeader("X-API-KEY")

		//если ключ пустой - отдаём ошибку, обозначающую, что необходимо ввести ключ
		if len(headerKey) < 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"Ошибка": "укажите заголовок с валидным API ключом"})
			c.Abort()
			return
		}

		//если полученный ключ не соответствует ключу из конфигурации - отдаём ошибку
		if headerKey != apiKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"Ошибка": "невалидный API ключ"})
			c.Abort()
			return
		}

		//если полученный ключ прошел обе проверки - позволяем выполнить дальнейшую логику программы
		c.Next()
	}
}
