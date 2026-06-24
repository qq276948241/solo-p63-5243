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

	appointment, err := h.service.CreateAppointment(userCtx.UserID, &req)
	if err != nil {
		common.Error(c, 400, err.Error())
		return
	}

	common.SuccessWithMsg(c, "预约成功", appointment)
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

	var appointments []AppointmentDetail
	var err error

	if userCtx.Role == user.RolePatient {
		appointments, err = h.service.GetPatientAppointments(userCtx.UserID, &req)
	} else if userCtx.Role == user.RoleDoctor {
		appointments, err = h.service.GetDoctorAppointments(userCtx.UserID, &req)
	} else {
		common.Forbidden(c, "角色错误")
		return
	}

	if err != nil {
		common.ServerError(c, "获取预约列表失败")
		return
	}

	common.Success(c, appointments)
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

	if req.Date == "" {
		req.Date = ""
	}

	appointments, err := h.service.GetDoctorAppointments(userCtx.UserID, &req)
	if err != nil {
		common.ServerError(c, "获取预约列表失败")
		return
	}

	common.Success(c, appointments)
}

func (h *Handler) CancelAppointment(c *gin.Context) {
	userCtx := common.GetCurrentUser(c)
	if userCtx == nil || userCtx.Role != user.RolePatient {
		common.Forbidden(c, "仅患者可取消预约")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		common.ParamError(c, "预约ID格式错误")
		return
	}

	if err := h.service.CancelAppointment(uint(id), userCtx.UserID); err != nil {
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

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		common.ParamError(c, "预约ID格式错误")
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	if err := h.service.CompleteAppointment(uint(id), userCtx.UserID, req.Remark); err != nil {
		common.Error(c, 400, err.Error())
		return
	}

	common.SuccessWithMsg(c, "标记就诊完成成功", nil)
}

func (h *Handler) GetAppointmentDetail(c *gin.Context) {
	userCtx := common.GetCurrentUser(c)
	if userCtx == nil {
		common.Unauthorized(c, "用户未认证")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		common.ParamError(c, "预约ID格式错误")
		return
	}

	appointment, err := h.service.GetAppointmentByID(uint(id))
	if err != nil {
		common.NotFound(c, "预约不存在")
		return
	}

	if appointment.PatientID != userCtx.UserID && appointment.DoctorID != userCtx.UserID {
		common.Forbidden(c, "无权查看他人预约")
		return
	}

	common.Success(c, appointment)
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	appointmentGroup := r.Group("/appointment")
	appointmentGroup.Use(common.JWTAuth())
	{
		appointmentGroup.POST("", common.RoleAuth(user.RolePatient), h.CreateAppointment)
		appointmentGroup.GET("/my", h.GetMyAppointments)
		appointmentGroup.GET("/:id", h.GetAppointmentDetail)
		appointmentGroup.POST("/:id/cancel", common.RoleAuth(user.RolePatient), h.CancelAppointment)
		appointmentGroup.POST("/:id/complete", common.RoleAuth(user.RoleDoctor), h.CompleteAppointment)
		appointmentGroup.GET("/doctor/today", common.RoleAuth(user.RoleDoctor), h.GetDoctorTodayAppointments)
	}
}
