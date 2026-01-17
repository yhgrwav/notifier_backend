package v1

import (
	"RedCollar/internal/domain"
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Описываем, что хендлер ждет от сервиса
type IncidentService interface {
	CheckLocation(ctx context.Context, req domain.LocationCheckRequest, limit, offset int) (domain.LocationCheckResponse, error)
	GetStats(ctx context.Context, statsTime int) ([]domain.StatisticResponse, error)
	Create(ctx context.Context, i *domain.Incident) (string, error)
	Get(ctx context.Context, lat, lon float64, limit, offset int) ([]*domain.Incident, error)
	GetByID(ctx context.Context, id string) (*domain.Incident, error)
	Update(ctx context.Context, id string, i *domain.Incident) (uuid.UUID, error)
	Delete(ctx context.Context, id string) error
}
type Handler struct {
	service   IncidentService
	statsTime int
}

func NewHandler(s IncidentService, st int) *Handler {
	return &Handler{
		service:   s,
		statsTime: st,
	}
}

// Init будет отвечать за регистрацию путей
func (h *Handler) Init(api *gin.RouterGroup) {
	v1 := api.Group("v1") //требуемый путь из ТЗ
	{
		//эндпоинт проверки координат для юзера
		v1.POST("/location/check", h.checkLocation)

		incidents := v1.Group("/incidents")

		//используем проверку на валидный ключ для группы эндпоинтов, которые использует оператор
		incidents.Use(middleware.MiddlewareAuth(apiKey))
		{
			//эндпоинт для получения статистики за n минут
			incidents.GET("/stats", h.GetStats)

			//CRUD для роли оператора по условиям ТЗ
			//сначала указываются статические эндпоинты, а затем динамические, чтобы не возникла проблема затенения
			//из-за специфики реализации роутинга групп эндпоинтов
			incidents.POST("/", h.CreateIncident)
			incidents.GET("/", h.GetIncidents)
			incidents.GET("/:id", h.GetIncidentByID)
			incidents.PUT("/:id", h.UpdateIncident)
			incidents.DELETE("/:id", h.DeleteIncident)
		}
		//health check по условию ТЗ
		v1.GET("/system/health", h.GetHealth)
	}
}

// Run отвечает за то, чтобы запустить http сервер на порту, который в main.go мы будем указывать из .env конфигурации
func (h *Handler) Run(port string) error {
	router := gin.Default()

	//инициализируем роутинг
	h.Init(router.Group("/api"))

	return router.Run(":" + port)
}
