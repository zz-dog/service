package handler

import (
	"errors"
	"net/http"
	"time"

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
	Phone    string `json:"phone" binding:"required,len=11"`
	Nickname string `json:"nickname" binding:"required,min=2,max=10" `
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

type getUserListRequest struct {
	Page     int `form:"page" binding:"required,min=1"`
	PageSize int `form:"page_size" binding:"required,min=1,max=100"`
}

func (h *Handler) GetUserList(c *gin.Context) {
	var req getUserListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return
	}

	users, total, err := h.userSvc.GetUserList(c.Request.Context(), req.Page, req.PageSize)
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.SuccessMsg(c, "getUserList", gin.H{
		"users": users,
		"total": total,
	})
}

// updateUserRequest 更新用户资料请求结构体
// 语义为整体替换：资料字段以本次提交为准，未提交的可选字段会被置空。
type updateUserRequest struct {
	Nickname string     `json:"nickname" binding:"required,min=2,max=10"`
	Phone    string     `json:"phone" binding:"omitempty,len=11"`
	Email    string     `json:"email" binding:"omitempty,email"`
	Avatar   string     `json:"avatar"`
	Gender   int8       `json:"gender" binding:"omitempty,min=0,max=2"`
	Birthday *time.Time `json:"birthday"`
}

// UpdateUser 更新当前登录用户的基础资料（用户 ID 取自 JWT，不信任客户端传入）
func (h *Handler) UpdateUser(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, http.StatusUnauthorized, "未登录或登录已过期")
		return
	}

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return
	}
	in := userapp.UpdateUserInput{
		Nickname: req.Nickname,
		Phone:    req.Phone,
		Email:    req.Email,
		Avatar:   req.Avatar,
		Gender:   req.Gender,
		Birthday: req.Birthday,
	}
	result, err := h.userSvc.UpdateUser(c.Request.Context(), userID, in)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.SuccessMsg(c, "updateUser", result)
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
	case errors.Is(err, domainuser.ErrInvalidGender):
		response.BadRequest(c, http.StatusBadRequest, err.Error())
	default:
		response.ServerError(c, http.StatusInternalServerError, err.Error())
	}
}
