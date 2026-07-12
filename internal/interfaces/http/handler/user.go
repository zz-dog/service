package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	userapp "github.com/wsc-zz/service/internal/application/user"
	domainuser "github.com/wsc-zz/service/internal/domain/user"
	"github.com/wsc-zz/service/pkg/response"
	"github.com/wsc-zz/service/pkg/validator"
)

// Handler 是用户相关的 HTTP 处理器，持有应用服务以处理请求。
type Handler struct {
	userSvc *userapp.Service
}

// NewHandler 构造处理器，注入用户应用服务。
func NewHandler(userSvc *userapp.Service) *Handler {
	return &Handler{userSvc: userSvc}
}

// registerRequest 注册请求结构体（带 gin binding 标签，仅接口层感知 Web 框架）
type registerRequest struct {
	Username string `json:"username" binding:"required,min=2,max=10"`
	Password string `json:"password" binding:"required,min=6,max=20"`
	Phone    string `json:"phone" binding:"len=11"`
	Nickname string `json:"nickname" binding:"min=2,max=10"`
}

// loginRequest 登录请求结构体
type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	IP       string `json:"ip"`
}

// Register 注册用户
func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return
	}
	in := userapp.RegisterInput{
		Username: req.Username,
		Password: req.Password,
		Phone:    req.Phone,
		Nickname: req.Nickname,
	}
	result, err := h.userSvc.Register(c.Request.Context(), in)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.SuccessMsg(c, "registerUser", result)
}

// Login 用户名密码登录
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return
	}
	in := userapp.LoginInput{
		Username: req.Username,
		Password: req.Password,
		IP:       req.IP,
	}
	resp, err := h.userSvc.Login(c.Request.Context(), in)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.SuccessMsg(c, "loginWithUsername", resp)
}

// writeError 将领域/应用错误映射为对应的 HTTP 响应。
func (h *Handler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainuser.ErrUserAlreadyExists):
		response.BadRequest(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, domainuser.ErrInvalidCredentials):
		response.Unauthorized(c, http.StatusUnauthorized, err.Error())
	case errors.Is(err, domainuser.ErrUserDisabled):
		response.Forbidden(c, http.StatusForbidden, err.Error())
	case errors.Is(err, domainuser.ErrUserNotFound):
		response.NotFound(c, http.StatusNotFound, err.Error())
	default:
		response.ServerError(c, http.StatusInternalServerError, err.Error())
	}
}
