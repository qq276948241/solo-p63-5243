package schedule

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

func (h *Handler) CreateSchedule(c *gin.Context) {
	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	schedule, err := h.service.CreateSchedule(&req)
	if err != nil {
		common.Error(c, 400, err.Error())
		return
	}

	common.Success(c, schedule)
}

func (h *Handler) GetScheduleList(c *gin.Context) {
	var req QueryScheduleRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	schedules, err := h.service.GetScheduleList(&req)
	if err != nil {
		common.ServerError(c, "获取排班列表失败")
		return
	}

	common.Success(c, schedules)
}

func (h *Handler) GetDoctorTodaySchedules(c *gin.Context) {
	userCtx := common.GetCurrentUser(c)
	if userCtx == nil || userCtx.Role != user.RoleDoctor {
		common.Forbidden(c, "仅医生可访问")
		return
	}

	date := c.Query("date")
	schedules, err := h.service.GetDoctorScheduleList(userCtx.UserID, date)
	if err != nil {
		common.ServerError(c, "获取排班列表失败")
		return
	}

	common.Success(c, schedules)
}

func (h *Handler) CancelSchedule(c *gin.Context) {
	userCtx := common.GetCurrentUser(c)
	if userCtx == nil || userCtx.Role != user.RoleDoctor {
		common.Forbidden(c, "仅医生可访问")
		return
	}

	scheduleIDStr := c.Param("id")
	scheduleID, err := strconv.ParseUint(scheduleIDStr, 10, 32)
	if err != nil {
		common.ParamError(c, "排班ID格式错误")
		return
	}

	if err := h.service.CancelSchedule(uint(scheduleID), userCtx.UserID); err != nil {
		common.Error(c, 400, err.Error())
		return
	}

	common.SuccessWithMsg(c, "停诊成功", nil)
}

func (h *Handler) GetScheduleDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		common.ParamError(c, "排班ID格式错误")
		return
	}

	schedule, err := h.service.GetScheduleByID(uint(id))
	if err != nil {
		common.NotFound(c, "排班不存在")
		return
	}

	common.Success(c, schedule)
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	scheduleGroup := r.Group("/schedule")
	scheduleGroup.Use(common.JWTAuth())
	{
		scheduleGroup.GET("", h.GetScheduleList)
		scheduleGroup.GET("/:id", h.GetScheduleDetail)
		scheduleGroup.POST("", common.RoleAuth(user.RoleDoctor), h.CreateSchedule)
		scheduleGroup.GET("/doctor/today", common.RoleAuth(user.RoleDoctor), h.GetDoctorTodaySchedules)
		scheduleGroup.POST("/:id/cancel", common.RoleAuth(user.RoleDoctor), h.CancelSchedule)
	}
}
