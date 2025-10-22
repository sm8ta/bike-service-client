package http

import (
	"net/http"
	"time"

	"github.com/sm8ta/webike_bike_microservice_nikita/internal/core/domain"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/core/ports"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/core/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ComponentHandler struct {
	componentService *services.ComponentService
	bikeService      *services.BikeService
	logger           ports.LoggerPort
	metrics          ports.MetricsPort
}

type ComponentRequest struct {
	BikeID           string `json:"bike_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name             string `json:"name" binding:"required" example:"handlebars"`
	Brand            string `json:"brand,omitempty" example:"Shimano"`
	Model            string `json:"model,omitempty" example:"Deore XT"`
	InstalledMileage int    `json:"installed_mileage" binding:"required" example:"1000"`
	MaxMileage       int    `json:"max_mileage" binding:"required" example:"5000"`
}

type UpdateComponent struct {
	Name             *string `json:"name,omitempty" example:"handlebars"`
	Brand            *string `json:"brand,omitempty" example:"Shimano"`
	Model            *string `json:"model,omitempty" example:"XT"`
	InstalledMileage *int    `json:"installed_mileage,omitempty" example:"1000"`
	MaxMileage       *int    `json:"max_mileage,omitempty" example:"5000"`
}

func NewComponentHandler(
	componentService *services.ComponentService,
	bikeService *services.BikeService,
	logger ports.LoggerPort,
	metrics ports.MetricsPort,
) *ComponentHandler {
	return &ComponentHandler{
		componentService: componentService,
		bikeService:      bikeService,
		logger:           logger,
		metrics:          metrics,
	}
}

// @Summary Создать компонент
// @Description Добавление компонента к байку
// @Tags components
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body ComponentRequest true "Данные компонента"
// @Success 201 {object} successResponse "Компонент создан"
// @Failure 400 {object} errorResponse "Неверный запрос"
// @Failure 401 {object} errorResponse "Не авторизован"
// @Failure 403 {object} errorResponse "Доступ запрещен"
// @Router /components [post]
func (h *ComponentHandler) CreateComponent(c *gin.Context) {
	start := time.Now()
	defer func() {
		h.metrics.RecordMetrics(c, start)
	}()

	payload, exists := getAuthPayload(c, "authorization_payload")
	if !exists {
		h.logger.Warn("Unauthorized access attempt to CreateComponent", map[string]interface{}{
			"ip": c.ClientIP(),
		})
		newErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req ComponentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed JSON parse in create component", map[string]interface{}{
			"error": err.Error(),
		})
		newErrorResponse(c, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	// смотрим че байк существует и принадлежит юзеру
	bike, err := h.bikeService.GetBikeByID(c.Request.Context(), req.BikeID)
	if err != nil {
		h.logger.Error("Failed to get bike", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": req.BikeID,
		})
		newErrorResponse(c, http.StatusNotFound, "Bike not found")
		return
	}

	if payload.Role != domain.Admin && payload.UserID != bike.UserID {
		h.logger.Warn("Access denied to add component", map[string]interface{}{
			"requester_id": payload.UserID.String(),
			"bike_owner":   bike.UserID.String(),
			"bike_id":      req.BikeID,
		})
		newErrorResponse(c, http.StatusForbidden, "Access denied")
		return
	}

	bikeUUID, err := uuid.Parse(req.BikeID)
	if err != nil {
		h.logger.Error("Invalid bike ID format", map[string]interface{}{
			"bike_id": req.BikeID,
		})
		newErrorResponse(c, http.StatusBadRequest, "Invalid bike ID")
		return
	}

	component := &domain.Component{
		BikeID:           bikeUUID,
		Name:             domain.ComponentName(req.Name),
		Brand:            req.Brand,
		Model:            req.Model,
		InstalledAt:      time.Now(),
		InstalledMileage: req.InstalledMileage,
		MaxMileage:       req.MaxMileage,
	}

	createdComponent, err := h.componentService.CreateComponent(c.Request.Context(), component)
	if err != nil {
		h.logger.Error("Failed to create component", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": req.BikeID,
		})
		newErrorResponse(c, http.StatusInternalServerError, "Failed to create component")
		return
	}

	h.logger.Info("Component created successfully", map[string]interface{}{
		"component_id": createdComponent.ID,
		"bike_id":      createdComponent.BikeID,
	})

	newSuccessResponse(c, http.StatusCreated, "Component created successfully", createdComponent)
}

// @Summary Получить компонент
// @Description Получение информации о компоненте по ID
// @Tags components
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "ID компонента" example:"jdk2-fsjmk-daslkdo2-321md-jsnlaljdn"
// @Success 200 {object} successResponse "Компонент найден"
// @Failure 401 {object} errorResponse "Не авторизован"
// @Failure 403 {object} errorResponse "Доступ запрещен"
// @Failure 404 {object} errorResponse "Компонент не найден"
// @Router /components/{id} [get]
func (h *ComponentHandler) GetComponent(c *gin.Context) {
	start := time.Now()
	defer func() {
		h.metrics.RecordMetrics(c, start)
	}()

	componentID := c.Param("id")

	payload, exists := getAuthPayload(c, "authorization_payload")
	if !exists {
		h.logger.Warn("Unauthorized access attempt to GetComponent", map[string]interface{}{
			"component_id": componentID,
			"ip":           c.ClientIP(),
		})
		newErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	component, err := h.componentService.GetComponentByID(c.Request.Context(), componentID)
	if err != nil {
		h.logger.Error("Failed to get component", map[string]interface{}{
			"error":        err.Error(),
			"component_id": componentID,
		})
		newErrorResponse(c, http.StatusNotFound, "Component not found")
		return
	}

	// смотрим че байк принадлежит юзеру
	bike, err := h.bikeService.GetBikeByID(c.Request.Context(), component.BikeID.String())
	if err != nil {
		h.logger.Error("Failed to get bike", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": component.BikeID,
		})
		newErrorResponse(c, http.StatusNotFound, "Bike not found")
		return
	}

	if payload.Role != domain.Admin && payload.UserID != bike.UserID {
		h.logger.Warn("Access denied to component", map[string]interface{}{
			"requester_id": payload.UserID.String(),
			"bike_owner":   bike.UserID.String(),
			"component_id": componentID,
		})
		newErrorResponse(c, http.StatusForbidden, "Access denied")
		return
	}

	newSuccessResponse(c, http.StatusOK, "Component found", component)
}

// @Summary Обновить компонент
// @Description Обновление данных компонента
// @Tags components
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "ID компонента" example:"jdk2-fsjmk-daslkdo2-321md-jsnlaljdn"
// @Param request body UpdateComponent true "Данные для обновления"
// @Success 200 {object} successResponse "Компонент обновлен"
// @Failure 400 {object} errorResponse "Неверный запрос"
// @Failure 401 {object} errorResponse "Не авторизован"
// @Failure 403 {object} errorResponse "Доступ запрещен"
// @Router /components/{id} [put]
func (h *ComponentHandler) UpdateComponent(c *gin.Context) {
	start := time.Now()
	defer func() {
		h.metrics.RecordMetrics(c, start)
	}()

	componentID := c.Param("id")

	payload, exists := getAuthPayload(c, "authorization_payload")
	if !exists {
		h.logger.Warn("Unauthorized access attempt to UpdateComponent", map[string]interface{}{
			"component_id": componentID,
			"ip":           c.ClientIP(),
		})
		newErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// смотрим че комп. существует
	existingComponent, err := h.componentService.GetComponentByID(c.Request.Context(), componentID)
	if err != nil {
		h.logger.Error("Failed to get component", map[string]interface{}{
			"error":        err.Error(),
			"component_id": componentID,
		})
		newErrorResponse(c, http.StatusNotFound, "Component not found")
		return
	}

	// смотрим че байк принадлежит юзеру
	bike, err := h.bikeService.GetBikeByID(c.Request.Context(), existingComponent.BikeID.String())
	if err != nil {
		h.logger.Error("Failed to get bike", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": existingComponent.BikeID,
		})
		newErrorResponse(c, http.StatusNotFound, "Bike not found")
		return
	}

	if payload.Role != domain.Admin && payload.UserID != bike.UserID {
		h.logger.Warn("Access denied to update component", map[string]interface{}{
			"requester_id": payload.UserID.String(),
			"bike_owner":   bike.UserID.String(),
			"component_id": componentID,
		})
		newErrorResponse(c, http.StatusForbidden, "Access denied")
		return
	}

	var req UpdateComponent
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed JSON parse in update component", map[string]interface{}{
			"error": err.Error(),
		})
		newErrorResponse(c, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	parsedID, err := uuid.Parse(componentID)
	if err != nil {
		h.logger.Error("Invalid component ID format", map[string]interface{}{
			"component_id": componentID,
		})
		newErrorResponse(c, http.StatusBadRequest, "Invalid component ID")
		return
	}

	component := &domain.Component{
		ID:     parsedID,
		BikeID: existingComponent.BikeID,
	}
	if req.Name != nil {
		component.Name = domain.ComponentName(*req.Name)
	}
	if req.Brand != nil {
		component.Brand = *req.Brand
	}
	if req.Model != nil {
		component.Model = *req.Model
	}
	if req.InstalledMileage != nil {
		component.InstalledMileage = *req.InstalledMileage
	}
	if req.MaxMileage != nil {
		component.MaxMileage = *req.MaxMileage
	}

	updatedComponent, err := h.componentService.UpdateComponent(c.Request.Context(), component)
	if err != nil {
		h.logger.Error("Failed to update component", map[string]interface{}{
			"error":        err.Error(),
			"component_id": componentID,
		})
		newErrorResponse(c, http.StatusInternalServerError, "Update failed")
		return
	}

	h.logger.Info("Component updated successfully", map[string]interface{}{
		"component_id": componentID,
	})

	newSuccessResponse(c, http.StatusOK, "Component updated successfully", updatedComponent)
}

// @Summary Удалить компонент
// @Description Удаление компонента
// @Tags components
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "ID компонента" example:"jdk2-fsjmk-daslkdo2-321md-jsnlaljdn"
// @Success 200 {object} successResponse "Компонент удален"
// @Failure 401 {object} errorResponse "Не авторизован"
// @Failure 403 {object} errorResponse "Доступ запрещен"
// @Router /components/{id} [delete]
func (h *ComponentHandler) DeleteComponent(c *gin.Context) {
	start := time.Now()
	defer func() {
		h.metrics.RecordMetrics(c, start)
	}()

	componentID := c.Param("id")

	payload, exists := getAuthPayload(c, "authorization_payload")
	if !exists {
		h.logger.Warn("Unauthorized access attempt to DeleteComponent", map[string]interface{}{
			"component_id": componentID,
			"ip":           c.ClientIP(),
		})
		newErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Смотри че компонент существует
	existingComponent, err := h.componentService.GetComponentByID(c.Request.Context(), componentID)
	if err != nil {
		h.logger.Error("Failed to get component", map[string]interface{}{
			"error":        err.Error(),
			"component_id": componentID,
		})
		newErrorResponse(c, http.StatusNotFound, "Component not found")
		return
	}

	// проверяем что байк принадлежит юзеру
	bike, err := h.bikeService.GetBikeByID(c.Request.Context(), existingComponent.BikeID.String())
	if err != nil {
		h.logger.Error("Failed to get bike", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": existingComponent.BikeID,
		})
		newErrorResponse(c, http.StatusNotFound, "Bike not found")
		return
	}

	if payload.Role != domain.Admin && payload.UserID != bike.UserID {
		h.logger.Warn("Access denied to delete component", map[string]interface{}{
			"requester_id": payload.UserID.String(),
			"bike_owner":   bike.UserID.String(),
			"component_id": componentID,
		})
		newErrorResponse(c, http.StatusForbidden, "Access denied")
		return
	}

	err = h.componentService.DeleteComponent(c.Request.Context(), componentID)
	if err != nil {
		h.logger.Error("Failed to delete component", map[string]interface{}{
			"error":        err.Error(),
			"component_id": componentID,
		})
		newErrorResponse(c, http.StatusInternalServerError, "Delete failed")
		return
	}

	h.logger.Info("Component deleted successfully", map[string]interface{}{
		"component_id": componentID,
	})

	newSuccessResponse(c, http.StatusOK, "Component deleted successfully", nil)
}
