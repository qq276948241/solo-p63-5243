package appointment

import (
	"clinic/internal/common"
	"clinic/internal/user"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ==================== 预约相关 Handler ====================

func (h *Handler) CreateAppointment(c *gin.Context) {
	userCtx := common.GetCurrentUser(c)
	if userCtx == nil || userCtx.Role != user.RolePatient {
		common.Forbidden(c, "仅患者可预约")
		return
	}

	var req CreateAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	apt, err := h.service.CreateAppointment(userCtx.UserID, &req)
	if err != nil {
		common.Error(c, 400, err.Error())
		return
	}

	common.SuccessWithMsg(c, "预约成功", apt)
}

func (h *Handler) GetMyAppointments(c *gin.Context) {
	userCtx := common.GetCurrentUser(c)
	if userCtx == nil {
		common.Unauthorized(c, "用户未认证")
		return
	}

	var req QueryAppointmentRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	var list []AppointmentDetail
	var err error

	switch userCtx.Role {
	case user.RolePatient:
		list, err = h.service.GetPatientAppointments(userCtx.UserID, &req)
	case user.RoleDoctor:
		list, err = h.service.GetDoctorAppointments(userCtx.UserID, &req)
	default:
		common.Forbidden(c, "角色错误")
		return
	}

	if err != nil {
		common.ServerError(c, "获取预约列表失败")
		return
	}

	common.Success(c, list)
}

func (h *Handler) GetAppointmentDetail(c *gin.Context) {
	userCtx := common.GetCurrentUser(c)
	if userCtx == nil {
		common.Unauthorized(c, "用户未认证")
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		common.ParamError(c, "预约ID格式错误")
		return
	}

	apt, err := h.service.GetAppointmentByID(id)
	if err != nil {
		common.NotFound(c, "预约不存在")
		return
	}

	if !apt.BelongsToPatient(userCtx.UserID) && !apt.BelongsToDoctor(userCtx.UserID) {
		common.Forbidden(c, "无权查看他人预约")
		return
	}

	common.Success(c, apt)
}

func (h *Handler) CancelAppointment(c *gin.Context) {
	userCtx := common.GetCurrentUser(c)
	if userCtx == nil || userCtx.Role != user.RolePatient {
		common.Forbidden(c, "仅患者可取消预约")
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		common.ParamError(c, "预约ID格式错误")
		return
	}

	if err := h.service.CancelAppointment(id, userCtx.UserID); err != nil {
		common.Error(c, 400, err.Error())
		return
	}

	common.SuccessWithMsg(c, "取消预约成功", nil)
}

func (h *Handler) CompleteAppointment(c *gin.Context) {
	userCtx := common.GetCurrentUser(c)
	if userCtx == nil || userCtx.Role != user.RoleDoctor {
		common.Forbidden(c, "仅医生可操作")
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		common.ParamError(c, "预约ID格式错误")
		return
	}

	var req CompleteAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	if err := h.service.CompleteAppointment(id, userCtx.UserID, req.Remark); err != nil {
		common.Error(c, 400, err.Error())
		return
	}

	common.SuccessWithMsg(c, "标记就诊完成成功", nil)
}

func (h *Handler) GetDoctorTodayAppointments(c *gin.Context) {
	userCtx := common.GetCurrentUser(c)
	if userCtx == nil || userCtx.Role != user.RoleDoctor {
		common.Forbidden(c, "仅医生可访问")
		return
	}

	var req QueryAppointmentRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	list, err := h.service.GetDoctorAppointments(userCtx.UserID, &req)
	if err != nil {
		common.ServerError(c, "获取预约列表失败")
		return
	}

	common.Success(c, list)
}

// ==================== 评价相关 Handler ====================

func (h *Handler) CreateReview(c *gin.Context) {
	userCtx := common.GetCurrentUser(c)
	if userCtx == nil || userCtx.Role != user.RolePatient {
		common.Forbidden(c, "仅患者可评价")
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		common.ParamError(c, "预约ID格式错误")
		return
	}

	var req CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	review, err := h.service.CreateReview(id, userCtx.UserID, &req)
	if err != nil {
		common.Error(c, 400, err.Error())
		return
	}

	common.SuccessWithMsg(c, "评价成功", review)
}

func (h *Handler) GetDoctorRating(c *gin.Context) {
	userCtx := common.GetCurrentUser(c)
	if userCtx == nil {
		common.Unauthorized(c, "用户未认证")
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		common.ParamError(c, "医生ID格式错误")
		return
	}

	rating, err := h.service.GetDoctorRating(id)
	if err != nil {
		common.ServerError(c, "获取评分失败")
		return
	}

	common.Success(c, rating)
}

func (h *Handler) GetDoctorReviews(c *gin.Context) {
	userCtx := common.GetCurrentUser(c)
	if userCtx == nil {
		common.Unauthorized(c, "用户未认证")
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		common.ParamError(c, "医生ID格式错误")
		return
	}

	list, err := h.service.GetDoctorReviews(id)
	if err != nil {
		common.ServerError(c, "获取评价列表失败")
		return
	}

	common.Success(c, list)
}

// ==================== 工具函数 ====================

func parseUintParam(c *gin.Context, key string) (uint, error) {
	val := c.Param(key)
	id, err := strconv.ParseUint(val, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

// ==================== 路由注册 ====================

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	api := r.Group("/appointment")
	api.Use(common.JWTAuth())
	{
		h.registerAppointmentRoutes(api)
		h.registerReviewRoutes(api)
	}
}

func (h *Handler) registerAppointmentRoutes(api *gin.RouterGroup) {
	patientOnly := common.RoleAuth(user.RolePatient)
	doctorOnly := common.RoleAuth(user.RoleDoctor)

	api.POST("", patientOnly, h.CreateAppointment)
	api.GET("/my", h.GetMyAppointments)
	api.GET("/:id", h.GetAppointmentDetail)
	api.POST("/:id/cancel", patientOnly, h.CancelAppointment)
	api.POST("/:id/complete", doctorOnly, h.CompleteAppointment)
	api.GET("/doctor/today", doctorOnly, h.GetDoctorTodayAppointments)
}

func (h *Handler) registerReviewRoutes(api *gin.RouterGroup) {
	patientOnly := common.RoleAuth(user.RolePatient)

	api.POST("/:id/review", patientOnly, h.CreateReview)
	api.GET("/doctor/:id/rating", h.GetDoctorRating)
	api.GET("/doctor/:id/reviews", h.GetDoctorReviews)
}
