package handle

import (
	"demo/internal/model"
	"demo/internal/service"
	"demo/pkg/response"
	"demo/pkg/validator"

	"github.com/gin-gonic/gin"
)

func RegisterUser(c *gin.Context) {
	var req model.RegisterUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 400, validator.ErrorMsg(err))
		return
	}
	userService := &service.UserService{}
	if err := userService.RegisterUser(req); err != nil {
		response.ServerError(c, 500, err.Error())
		return
	}
	response.SuccessMsg(c, "registerUser", req)
}

func LoginWithUsername(c *gin.Context) {
	var req model.LoginUserWithUsernameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 400, validator.ErrorMsg(err))
		return
	}
	userService := &service.UserService{}
	resp, err := userService.LoginWithUsername(req)
	if err != nil {
		response.Unauthorized(c, 401, err.Error())
		return
	}
	response.SuccessMsg(c, "loginWithUsername", resp)
}
