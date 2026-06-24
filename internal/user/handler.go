package user

import (
	"clinic/internal/common"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	user, err := h.service.Register(&req)
	if err != nil {
		common.Error(c, 400, err.Error())
		return
	}

	common.Success(c, user)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	resp, err := h.service.Login(&req)
	if err != nil {
		common.Error(c, 400, err.Error())
		return
	}

	common.Success(c, resp)
}

func (h *Handler) GetCurrentUser(c *gin.Context) {
	userCtx := common.GetCurrentUser(c)
	if userCtx == nil {
		common.Unauthorized(c, "用户未认证")
		return
	}

	user, err := h.service.GetUserByID(userCtx.UserID)
	if err != nil {
		common.ServerError(c, "获取用户信息失败")
		return
	}

	common.Success(c, user)
}

func (h *Handler) GetDoctorList(c *gin.Context) {
	doctors, err := h.service.GetDoctorList()
	if err != nil {
		common.ServerError(c, "获取医生列表失败")
		return
	}

	common.Success(c, doctors)
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	userGroup := r.Group("/user")
	{
		userGroup.POST("/register", h.Register)
		userGroup.POST("/login", h.Login)
		userGroup.GET("/me", common.JWTAuth(), h.GetCurrentUser)
		userGroup.GET("/doctors", common.JWTAuth(), h.GetDoctorList)
	}
}
