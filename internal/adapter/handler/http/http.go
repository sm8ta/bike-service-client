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

type CreateBikeResponse struct {
	BikeID    uuid.UUID `json:"bike_id"`
	UserID    uuid.UUID `json:"user_id"`
	BikeName  string    `json:"bike_name"`
	Model     string    `json:"model"`
	Type      string    `json:"type"`
	Year      int       `json:"year"`
	Mileage   int       `json:"mileage"`
	CreatedAt time.Time `json:"created_at"`
}

type GetBikeResponse struct {
	BikeID    uuid.UUID `json:"bike_id"`
	UserID    uuid.UUID `json:"user_id"`
	BikeName  string    `json:"bike_name"`
	Model     string    `json:"model"`
	Type      string    `json:"type"`
	Year      int       `json:"year"`
	Mileage   int       `json:"mileage"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type GetMyBikesResponse struct {
	Bikes []BikeInfo `json:"bikes"`
	Count int        `json:"count"`
}

type BikeInfo struct {
	BikeID    uuid.UUID `json:"bike_id"`
	UserID    uuid.UUID `json:"user_id"`
	BikeName  string    `json:"bike_name"`
	Model     string    `json:"model"`
	Type      string    `json:"type"`
	Year      int       `json:"year"`
	Mileage   int       `json:"mileage"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UpdateBikeResponse struct {
	BikeID    uuid.UUID `json:"bike_id"`
	UserID    uuid.UUID `json:"user_id"`
	BikeName  string    `json:"bike_name"`
	Model     string    `json:"model"`
	Type      string    `json:"type"`
	Year      int       `json:"year"`
	Mileage   int       `json:"mileage"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DeleteBikeResponse struct {
	Message string `json:"message"`
}

type GetBikeWithComponentsResponse struct {
	BikeID     uuid.UUID       `json:"bike_id"`
	UserID     uuid.UUID       `json:"user_id"`
	BikeName   string          `json:"bike_name"`
	Model      string          `json:"model"`
	Type       string          `json:"type"`
	Year       int             `json:"year"`
	Mileage    int             `json:"mileage"`
	Components []ComponentInfo `json:"components"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type ComponentInfo struct {
	ID               uuid.UUID `json:"id"`
	BikeID           uuid.UUID `json:"bike_id"`
	Name             string    `json:"name"`
	Brand            string    `json:"brand"`
	Model            string    `json:"model"`
	InstalledAt      time.Time `json:"installed_at"`
	InstalledMileage int       `json:"installed_mileage"`
	MaxMileage       int       `json:"max_mileage"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type UserResponseInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	DateOfBirth string    `json:"date_of_birth"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type GetBikeWithUserResponse struct {
	BikeID    uuid.UUID         `json:"bike_id"`
	UserID    uuid.UUID         `json:"user_id"`
	BikeName  string            `json:"bike_name"`
	Model     string            `json:"model"`
	Type      string            `json:"type"`
	Year      int               `json:"year"`
	Mileage   int               `json:"mileage"`
	User      *UserResponseInfo `json:"user,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
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
// @Success 201 {object} CreateBikeResponse "Байк создан"
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

	response := CreateBikeResponse{
		BikeID:    createdBike.BikeID,
		UserID:    createdBike.UserID,
		BikeName:  createdBike.BikeName,
		Model:     createdBike.Model,
		Type:      string(createdBike.Type),
		Year:      createdBike.Year,
		Mileage:   createdBike.Mileage,
		CreatedAt: createdBike.CreatedAt,
	}

	c.JSON(http.StatusCreated, response)
}

// @Summary Получить байк
// @Description Получение информации о байке по ID
// @Tags bikes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "ID байка" example:"jdk2-fsjmk-daslkdo2-321md-jsnlaljdn"
// @Success 200 {object} GetBikeResponse "Байк найден"
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
	if payload.Role != domain.Admin && payload.UserID != bike.UserID {
		h.logger.Warn("Access denied to bike", map[string]interface{}{
			"requester_id": payload.UserID.String(),
			"bike_owner":   bike.UserID.String(),
			"bike_id":      bikeID,
		})
		newErrorResponse(c, http.StatusForbidden, "Access denied")
		return
	}
	response := GetBikeResponse{
		BikeID:    bike.BikeID,
		UserID:    bike.UserID,
		BikeName:  bike.BikeName,
		Model:     bike.Model,
		Type:      string(bike.Type),
		Year:      bike.Year,
		Mileage:   bike.Mileage,
		CreatedAt: bike.CreatedAt,
		UpdatedAt: bike.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Получить байки пользователя по айди пользователя
// @Description Получение всех байков авторизованного пользователя
// @Tags bikes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} GetMyBikesResponse "Список байков пользователя"
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
	bikeInfos := make([]BikeInfo, len(bikes))
	for i, bike := range bikes {
		bikeInfos[i] = BikeInfo{
			BikeID:    bike.BikeID,
			UserID:    bike.UserID,
			BikeName:  bike.BikeName,
			Model:     bike.Model,
			Type:      string(bike.Type),
			Year:      bike.Year,
			Mileage:   bike.Mileage,
			CreatedAt: bike.CreatedAt,
			UpdatedAt: bike.UpdatedAt,
		}
	}

	response := GetMyBikesResponse{
		Bikes: bikeInfos,
		Count: len(bikeInfos),
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Обновить байк
// @Description Обновление данных байка
// @Tags bikes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "ID байка" example:"jdk2-fsjmk-daslkdo2-321md-jsnlaljdn"
// @Param request body UpdateBike true "Данные для обновления"
// @Success 200 {object} UpdateBikeResponse "Байк обновлен"
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
	response := UpdateBikeResponse{
		BikeID:    updatedBike.BikeID,
		UserID:    updatedBike.UserID,
		BikeName:  updatedBike.BikeName,
		Model:     updatedBike.Model,
		Type:      string(updatedBike.Type),
		Year:      updatedBike.Year,
		Mileage:   updatedBike.Mileage,
		UpdatedAt: updatedBike.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Удалить байк
// @Description Удаление байка
// @Tags bikes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "ID байка" example:"jdk2-fsjmk-daslkdo2-321md-jsnlaljdn"
// @Success 200 {object} DeleteBikeResponse "Байк удален"
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

	c.JSON(http.StatusOK, DeleteBikeResponse{
		Message: "Bike deleted successfully",
	})
}

// @Summary Получить байк с компонентами
// @Description Получение байка со всеми компонентами
// @Tags bikes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "ID байка" example:"jdk2-fsjmk-daslkdo2-321md-jsnlaljdn"
// @Success 200 {object} GetBikeWithComponentsResponse "Байк с компонентами"
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
	componentInfos := make([]ComponentInfo, len(bike.Components))
	for i, comp := range bike.Components {
		componentInfos[i] = ComponentInfo{
			ID:               comp.ID,
			BikeID:           comp.BikeID,
			Name:             string(comp.Name),
			Brand:            comp.Brand,
			Model:            comp.Model,
			InstalledAt:      comp.InstalledAt,
			InstalledMileage: comp.InstalledMileage,
			MaxMileage:       comp.MaxMileage,
			CreatedAt:        comp.CreatedAt,
			UpdatedAt:        comp.UpdatedAt,
		}
	}

	response := GetBikeWithComponentsResponse{
		BikeID:     bike.BikeID,
		UserID:     bike.UserID,
		BikeName:   bike.BikeName,
		Model:      bike.Model,
		Type:       string(bike.Type),
		Year:       bike.Year,
		Mileage:    bike.Mileage,
		Components: componentInfos,
		CreatedAt:  bike.CreatedAt,
		UpdatedAt:  bike.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Получить байк с пользователем
// @Description Получение информации о байке и его владельце
// @Tags bikes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "ID байка" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"
// @Success 200 {object} GetBikeWithUserResponse "Байк с пользователем"
// @Failure 401 {object} errorResponse "Не авторизован"
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
		h.logger.Warn("Unauthorized access attempt", map[string]interface{}{
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
		h.logger.Warn("Access denied", map[string]interface{}{
			"requester_id": payload.UserID.String(),
			"owner_id":     bike.UserID.String(),
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

	var userInfo *UserResponseInfo

	resp, err := h.userClient.Users.GetUsersID(params, authInfo)
	if err != nil {
		h.logger.Warn("Failed to get user from user-service", map[string]interface{}{
			"error":   err.Error(),
			"user_id": bike.UserID.String(),
		})
		userInfo = nil
	} else if resp != nil && resp.Payload != nil {
		// Маппинг из user_models.HTTPGetUserResponse в UserResponseInfo
		userInfo = &UserResponseInfo{
			ID:          resp.Payload.ID,
			Name:        resp.Payload.Name,
			Email:       resp.Payload.Email,
			DateOfBirth: resp.Payload.DateOfBirth,
			Role:        resp.Payload.Role,
			CreatedAt:   resp.Payload.CreatedAt,
			UpdatedAt:   resp.Payload.UpdatedAt,
		}
	}

	response := GetBikeWithUserResponse{
		BikeID:    bike.BikeID,
		UserID:    bike.UserID,
		BikeName:  bike.BikeName,
		Model:     bike.Model,
		Type:      string(bike.Type),
		Year:      bike.Year,
		Mileage:   bike.Mileage,
		User:      userInfo, // ← используем свою структуру
		CreatedAt: bike.CreatedAt,
		UpdatedAt: bike.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}
