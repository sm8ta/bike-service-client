package http

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/google/uuid"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/core/domain"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/core/ports"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/core/services"
	"github.com/sm8ta/webike_user_microservice_nikita/models"
	user_client "github.com/sm8ta/webike_user_microservice_nikita/pkg/client"

	"github.com/sm8ta/webike_user_microservice_nikita/pkg/client/users"
)

type BikeHandler struct {
	bikeService *services.BikeService
	logger      ports.LoggerPort
	metrics     ports.MetricsPort
	userClient  *user_client.UserMicroservice
}

type BikeRequest struct {
	Model   string `json:"model" binding:"required" example:"Mountain Bike Pro"`
	Type    string `json:"type" binding:"required" example:"mountain"`
	Mileage int    `json:"mileage" binding:"required" example:"1500"`
}

type UpdateBike struct {
	Model   *string `json:"model,omitempty" example:"New Model"`
	Type    *string `json:"type,omitempty" example:"mountain"`
	Mileage *int    `json:"mileage,omitempty" example:"2000"`
}

type BikeWithUserResponse struct {
	BikeID  string `json:"bike_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Model   string `json:"model" example:"Mountain Bike Pro"`
	Mileage int    `json:"mileage" example:"1500"`
	User    *models.HTTPSuccessResponse
}

func NewBikeHandler(
	bikeService *services.BikeService,
	logger ports.LoggerPort,
	metrics ports.MetricsPort,
	userClient *user_client.UserMicroservice,
) *BikeHandler {
	return &BikeHandler{
		bikeService: bikeService,
		logger:      logger,
		metrics:     metrics,
		userClient:  userClient,
	}
}

// @Summary Создать байк
// @Description Создание нового байка
// @Tags bikes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body BikeRequest true "Данные байка"
// @Success 201 {object} successResponse "Байк создан"
// @Failure 400 {object} errorResponse "Неверный запрос"
// @Failure 401 {object} errorResponse "Не авторизован"
// @Router /bikes [post]
func (h *BikeHandler) CreateBike(c *gin.Context) {
	start := time.Now()
	defer func() {
		h.metrics.RecordMetrics(c, start)
	}()

	payload, exists := getAuthPayload(c, "authorization_payload")
	if !exists {
		h.logger.Warn("Unauthorized access attempt to CreateBike", map[string]interface{}{
			"ip": c.ClientIP(),
		})
		newErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req BikeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed JSON parse in create bike", map[string]interface{}{
			"error": err.Error(),
		})
		newErrorResponse(c, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	bike := &domain.Bike{
		UserID:  payload.UserID,
		Model:   req.Model,
		Type:    domain.BikeType(req.Type),
		Mileage: req.Mileage,
	}

	createdBike, err := h.bikeService.CreateBike(c.Request.Context(), bike)
	if err != nil {
		h.logger.Error("Failed to create bike", map[string]interface{}{
			"error":   err.Error(),
			"user_id": payload.UserID,
		})
		newErrorResponse(c, http.StatusInternalServerError, "Failed to create bike")
		return
	}

	h.logger.Info("Bike created successfully", map[string]interface{}{
		"bike_id": createdBike.BikeID,
		"user_id": createdBike.UserID,
	})

	newSuccessResponse(c, http.StatusCreated, "Bike created successfully", createdBike)
}

// @Summary Получить байк
// @Description Получение информации о байке по ID
// @Tags bikes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "ID байка" example:"jdk2-fsjmk-daslkdo2-321md-jsnlaljdn"
// @Success 200 {object} successResponse "Байк найден"
// @Failure 401 {object} errorResponse "Не авторизован"
// @Failure 403 {object} errorResponse "Доступ запрещен"
// @Failure 404 {object} errorResponse "Байк не найден"
// @Router /bikes/{id} [get]
func (h *BikeHandler) GetBike(c *gin.Context) {
	start := time.Now()
	defer func() {
		h.metrics.RecordMetrics(c, start)
	}()

	bikeID := c.Param("id")

	payload, exists := getAuthPayload(c, "authorization_payload")
	if !exists {
		h.logger.Warn("Unauthorized access attempt to GetBike", map[string]interface{}{
			"bike_id": bikeID,
			"ip":      c.ClientIP(),
		})
		newErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	bike, err := h.bikeService.GetBikeByID(c.Request.Context(), bikeID)
	if err != nil {
		h.logger.Error("Failed to get bike", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": bikeID,
		})
		newErrorResponse(c, http.StatusNotFound, "Bike not found")
		return
	}
	// проверка на админа или владельца
	if payload.Role != domain.Admin && payload.UserID != bike.UserID {
		h.logger.Warn("Access denied to bike", map[string]interface{}{
			"requester_id": payload.UserID.String(),
			"bike_owner":   bike.UserID.String(),
			"bike_id":      bikeID,
		})
		newErrorResponse(c, http.StatusForbidden, "Access denied")
		return
	}

	newSuccessResponse(c, http.StatusOK, "Bike found", bike)
}

// @Summary Получить байки пользователя по айди пользователя
// @Description Получение всех байков авторизованного пользователя
// @Tags bikes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} successResponse "Список байков пользователя"
// @Failure 401 {object} errorResponse "Не авторизован"
// @Failure 500 {object} errorResponse "Внутренняя ошибка сервера"
// @Router /bikes/my [get]
func (h *BikeHandler) GetMyBikes(c *gin.Context) {
	start := time.Now()
	defer func() {
		h.metrics.RecordMetrics(c, start)
	}()

	payload, exists := getAuthPayload(c, "authorization_payload")
	if !exists {
		h.logger.Warn("Unauthorized access attempt to GetMyBikes", map[string]interface{}{
			"ip": c.ClientIP(),
		})
		newErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	bikes, err := h.bikeService.GetBikesByUserID(c.Request.Context(), payload.UserID.String())
	if err != nil {
		h.logger.Error("Failed to get bikes", map[string]interface{}{
			"error":   err.Error(),
			"user_id": payload.UserID,
		})
		newErrorResponse(c, http.StatusInternalServerError, "Failed to get bikes")
		return
	}

	newSuccessResponse(c, http.StatusOK, "Bikes retrieved successfully", bikes)
}

// @Summary Обновить байк
// @Description Обновление данных байка
// @Tags bikes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "ID байка" example:"jdk2-fsjmk-daslkdo2-321md-jsnlaljdn"
// @Param request body UpdateBike true "Данные для обновления"
// @Success 200 {object} domain.Bike "Байк обновлен"
// @Failure 400 {object} errorResponse "Неверный запрос"
// @Failure 401 {object} errorResponse "Не авторизован"
// @Failure 403 {object} errorResponse "Доступ запрещен"
// @Router /bikes/{id} [put]
func (h *BikeHandler) UpdateBike(c *gin.Context) {
	start := time.Now()
	defer func() {
		h.metrics.RecordMetrics(c, start)
	}()

	bikeID := c.Param("id")

	payload, exists := getAuthPayload(c, "authorization_payload")
	if !exists {
		h.logger.Warn("Unauthorized access attempt to UpdateBike", map[string]interface{}{
			"bike_id": bikeID,
			"ip":      c.ClientIP(),
		})
		newErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	existingBike, err := h.bikeService.GetBikeByID(c.Request.Context(), bikeID)
	if err != nil {
		h.logger.Error("Failed to get bike", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": bikeID,
		})
		newErrorResponse(c, http.StatusNotFound, "Bike not found")
		return
	}

	if payload.Role != domain.Admin && payload.UserID != existingBike.UserID {
		h.logger.Warn("Access denied to update bike", map[string]interface{}{
			"requester_id": payload.UserID.String(),
			"bike_owner":   existingBike.UserID.String(),
			"bike_id":      bikeID,
		})
		newErrorResponse(c, http.StatusForbidden, "Access denied")
		return
	}

	var req UpdateBike
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed JSON parse in update bike", map[string]interface{}{
			"error": err.Error(),
		})
		newErrorResponse(c, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	parsedID, err := uuid.Parse(bikeID)
	if err != nil {
		h.logger.Error("Invalid bike ID format", map[string]interface{}{
			"bike_id": bikeID,
		})
		newErrorResponse(c, http.StatusBadRequest, "Invalid bike ID")
		return
	}

	bike := &domain.Bike{
		BikeID: parsedID,
		UserID: existingBike.UserID,
	}
	if req.Model != nil {
		bike.Model = *req.Model
	}
	if req.Type != nil {
		bikeType := domain.BikeType(*req.Type)
		bike.Type = bikeType
	}
	if req.Mileage != nil {
		bike.Mileage = *req.Mileage
	}

	updatedBike, err := h.bikeService.UpdateBike(c.Request.Context(), bike)
	if err != nil {
		h.logger.Error("Failed to update bike", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": bikeID,
		})
		newErrorResponse(c, http.StatusInternalServerError, "Update failed")
		return
	}

	h.logger.Info("Bike updated successfully", map[string]interface{}{
		"bike_id": bikeID,
	})

	newSuccessResponse(c, http.StatusOK, "Bike updated successfully", updatedBike)
}

// @Summary Удалить байк
// @Description Удаление байка
// @Tags bikes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "ID байка" example:"jdk2-fsjmk-daslkdo2-321md-jsnlaljdn"
// @Success 200 {object} successResponse "Байк удален"
// @Failure 401 {object} errorResponse "Не авторизован"
// @Failure 403 {object} errorResponse "Доступ запрещен"
// @Router /bikes/{id} [delete]
func (h *BikeHandler) DeleteBike(c *gin.Context) {
	start := time.Now()
	defer func() {
		h.metrics.RecordMetrics(c, start)
	}()

	bikeID := c.Param("id")

	payload, exists := getAuthPayload(c, "authorization_payload")
	if !exists {
		h.logger.Warn("Unauthorized access attempt to DeleteBike", map[string]interface{}{
			"bike_id": bikeID,
			"ip":      c.ClientIP(),
		})
		newErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	existingBike, err := h.bikeService.GetBikeByID(c.Request.Context(), bikeID)
	if err != nil {
		h.logger.Error("Failed to get bike", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": bikeID,
		})
		newErrorResponse(c, http.StatusNotFound, "Bike not found")
		return
	}

	if payload.Role != domain.Admin && payload.UserID != existingBike.UserID {
		h.logger.Warn("Access denied to delete bike", map[string]interface{}{
			"requester_id": payload.UserID.String(),
			"bike_owner":   existingBike.UserID.String(),
			"bike_id":      bikeID,
		})
		newErrorResponse(c, http.StatusForbidden, "Access denied")
		return
	}

	err = h.bikeService.DeleteBike(c.Request.Context(), bikeID)
	if err != nil {
		h.logger.Error("Failed to delete bike", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": bikeID,
		})
		newErrorResponse(c, http.StatusInternalServerError, "Delete failed")
		return
	}

	h.logger.Info("Bike deleted successfully", map[string]interface{}{
		"bike_id": bikeID,
	})

	newSuccessResponse(c, http.StatusOK, "Bike deleted successfully", nil)
}

// @Summary Получить байк с компонентами
// @Description Получение байка со всеми компонентами
// @Tags bikes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "ID байка" example:"jdk2-fsjmk-daslkdo2-321md-jsnlaljdn"
// @Success 200 {object} successResponse "Байк с компонентами"
// @Failure 401 {object} errorResponse "Не авторизован"
// @Failure 403 {object} errorResponse "Доступ запрещен"
// @Failure 404 {object} errorResponse "Байк не найден"
// @Router /bikes/{id}/with-components [get]
func (h *BikeHandler) GetBikeWithComponents(c *gin.Context) {
	start := time.Now()
	defer func() {
		h.metrics.RecordMetrics(c, start)
	}()

	bikeID := c.Param("id")

	payload, exists := getAuthPayload(c, "authorization_payload")
	if !exists {
		h.logger.Warn("Unauthorized access attempt to GetBikeWithComponents", map[string]interface{}{
			"bike_id": bikeID,
			"ip":      c.ClientIP(),
		})
		newErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	bike, err := h.bikeService.GetBikeWithComponents(c.Request.Context(), bikeID)
	if err != nil {
		h.logger.Error("Failed to get bike with components", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": bikeID,
		})
		newErrorResponse(c, http.StatusNotFound, "Bike not found")
		return
	}

	if payload.Role != domain.Admin && payload.UserID != bike.UserID {
		h.logger.Warn("Access denied to bike", map[string]interface{}{
			"requester_id": payload.UserID.String(),
			"bike_owner":   bike.UserID.String(),
			"bike_id":      bikeID,
		})
		newErrorResponse(c, http.StatusForbidden, "Access denied")
		return
	}

	newSuccessResponse(c, http.StatusOK, "Bike with components found", bike)
}

// @Summary Получить байк с пользователем
// @Description Получение информации о байке и его владельце
// @Tags bikes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "ID байка" example:"jdk2-fsjmk-daslkdo2-321md-jsnlaljdn"
// @Success 200 {object} successResponse "Байк с пользователем"
// @Failure 401 {object} errorResponse "Не авторизован"
// @Failure 403 {object} errorResponse "Доступ запрещен"
// @Failure 404 {object} errorResponse "Байк не найден"
// @Router /bikes/{id}/with-user [get]
func (h *BikeHandler) GetBikeWithUser(c *gin.Context) {
	start := time.Now()
	defer func() {
		h.metrics.RecordMetrics(c, start)
	}()

	bikeID := c.Param("id")

	payload, exists := getAuthPayload(c, "authorization_payload")
	if !exists {
		h.logger.Warn("Unauthorized access attempt to GetBikeWithUser", map[string]interface{}{
			"bike_id": bikeID,
			"ip":      c.ClientIP(),
		})
		newErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	bike, err := h.bikeService.GetBikeByID(c.Request.Context(), bikeID)
	if err != nil {
		h.logger.Error("Failed to get bike", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": bikeID,
		})
		newErrorResponse(c, http.StatusNotFound, "Bike not found")
		return
	}

	if payload.Role != domain.Admin && payload.UserID != bike.UserID {
		h.logger.Warn("Access denied to bike", map[string]interface{}{
			"requester_id": payload.UserID.String(),
			"bike_owner":   bike.UserID.String(),
			"bike_id":      bikeID,
		})
		newErrorResponse(c, http.StatusForbidden, "Access denied")
		return
	}

	params := users.NewGetUsersIDParams()
	params.ID = bike.UserID.String()
	params.Context = c.Request.Context()

	authHeader := c.GetHeader("Authorization")
	var authInfo runtime.ClientAuthInfoWriter
	if authHeader != "" {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		authInfo = httptransport.BearerToken(token)
	}

	userResp, err := h.userClient.Users.GetUsersID(params, authInfo)

	var user *models.HTTPSuccessResponse
	if err != nil {
		h.logger.Warn("Failed to get user from User service", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": bikeID,
			"user_id": bike.UserID.String(),
		})
		user = nil
	} else {
		user = userResp.Payload
	}

	response := BikeWithUserResponse{
		BikeID:  bike.BikeID.String(),
		Model:   bike.Model,
		Mileage: bike.Mileage,
		User:    user,
	}

	newSuccessResponse(c, http.StatusOK, "Bike with user found", response)
}
