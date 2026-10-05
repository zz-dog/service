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
}

// loginRequest 登录请求结构体
type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	IP       string `json:"ip"`
}

// Register 注册用户
//
//	@Summary		注册用户
//	@Description	用户名密码注册。用户名已存在时返回 400
//	@Tags			用户
//	@Accept			json
//	@Produce		json
//	@Param			request	body		registerRequest	true	"注册信息"
//	@Success		200		{object}	response.Response{data=userapp.UserDTO}
//	@Failure		400		{object}	response.Response	"参数校验失败或用户名已存在"
//	@Failure		500		{object}	response.Response	"服务器内部错误"
//	@Router			/user/register [post]
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
	}
	result, err := h.userSvc.Register(c.Request.Context(), in)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.SuccessMsg(c, "registerUser", result)
}

// Login 用户名密码登录
//
//	@Summary		用户名密码登录
//	@Description	登录成功返回 JWT token 与用户信息；用户名或密码错误返回 401（不区分账号是否存在），账号被禁用返回 403
//	@Tags			用户
//	@Accept			json
//	@Produce		json
//	@Param			request	body		loginRequest	true	"登录信息"
//	@Success		200		{object}	response.Response{data=userapp.LoginResult}
//	@Failure		400		{object}	response.Response	"参数校验失败"
//	@Failure		401		{object}	response.Response	"用户名或密码错误"
//	@Failure		403		{object}	response.Response	"账号已被禁用"
//	@Failure		500		{object}	response.Response	"服务器内部错误"
//	@Router			/user/login [post]
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
	Page     int    `form:"page" binding:"required,min=1"`
	PageSize int    `form:"pageSize" binding:"required,min=1,max=100"`
	UserID   uint   `form:"userId" `
	Username string `form:"username"`
}

// GetUserList 分页查询用户列表
//
//	@Summary		分页查询用户列表
//	@Description	分页查询用户，支持按用户ID精确过滤；需要登录（JWT）
//	@Tags			用户
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			page		query		int		true	"页码（从1开始）"
//	@Param			pageSize	query		int		true	"每页数量（1~100）"
//	@Param			userId		query		int		false	"用户ID（精确匹配）"
//	@Param			username	query		string	false	"用户名"
//	@Success		200			{object}	response.Response{data=userapp.GetUserListResult}
//	@Failure		400			{object}	response.Response	"参数校验失败"
//	@Failure		401			{object}	response.Response	"未登录或 token 失效"
//	@Failure		500			{object}	response.Response	"服务器内部错误"
//	@Router			/user/list [get]
func (h *Handler) GetUserList(c *gin.Context) {
	var req getUserListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return
	}

	result, err := h.userSvc.GetUserList(c.Request.Context(), userapp.GetUserListInput{
		Page:     req.Page,
		PageSize: req.PageSize,
		UserID:   req.UserID,
		Username: req.Username,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.SuccessMsg(c, "getUserList", result)
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
//
//	@Summary		更新当前用户资料
//	@Description	更新 JWT 对应用户的基础资料，用户ID取自 token。整体替换语义：资料字段以本次提交为准，未提交的可选字段（phone/email/avatar/gender/birthday）会被置空。需要登录（JWT）
//	@Tags			用户
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			request	body		updateUserRequest	true	"用户资料"
//	@Success		200		{object}	response.Response{data=userapp.UserDTO}
//	@Failure		400		{object}	response.Response	"参数校验失败或性别取值不合法"
//	@Failure		401		{object}	response.Response	"未登录或 token 失效"
//	@Failure		404		{object}	response.Response	"用户不存在"
//	@Failure		500		{object}	response.Response	"服务器内部错误"
//	@Router			/user/user/profile [put]
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

type changeStatusRequest struct {
	// int 而非 domainuser.Status：0 是合法取值（禁用），required 会拒绝 0；
	// 用 min/max 限定 0~1，swag 也能正确生成 schema
	Status int  `json:"status" binding:"min=0,max=1"`
	UserID uint `json:"userId" binding:"required"`
}

// ChangeStatus 修改用户状态
//
//	@Summary		修改用户状态
//	@Description	启用或禁用指定用户。status：0-禁用，1-正常。需要登录（JWT）
//	@Tags			用户
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			request	body		changeStatusRequest	true	"用户ID与目标状态"
//	@Success		200		{object}	response.Response
//	@Failure		400		{object}	response.Response	"参数校验失败"
//	@Failure		401		{object}	response.Response	"未登录或 token 失效"
//	@Failure		404		{object}	response.Response	"用户不存在"
//	@Failure		500		{object}	response.Response	"服务器内部错误"
//	@Router			/user/status [put]
func (h *Handler) ChangeStatus(c *gin.Context) {
	var req changeStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return
	}
	err := h.userSvc.ChangeStatus(c.Request.Context(), req.UserID, domainuser.Status(req.Status))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.SuccessMsg(c, "changeStatus", nil)
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

func (h *Handler) Logout(c *gin.Context) {
	jti, _ := c.Get("jti")
	exp, _ := c.Get("tokenExp")
	jtiStr, _ := jti.(string)
	expAt, _ := exp.(time.Time)
	if err := h.userSvc.Logout(c.Request.Context(), jtiStr, expAt); err != nil {
		response.ServerError(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.SuccessMsg(c, "logout", nil)
}
