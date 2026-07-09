package handle

import (
	"demo/internal/model"
	"demo/internal/service"
	"demo/pkg/response"
	"demo/pkg/validator"

	"github.com/gin-gonic/gin"
)

func CreateUser(c *gin.Context) {
	var req model.CreateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, validator.ErrorMsg(err))
		return
	}
	userService := &service.UserService{}
	err := userService.CreateUser(req)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.SuccessMsg(c, "createUser", req)
}

func LoginWithUsername(c *gin.Context) {
	var req model.LoginUserWithUsernameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, validator.ErrorMsg(err))
		return
	}
	userService := &service.UserService{}
	token, err := userService.LoginWithUsername(req)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.SuccessMsg(c, "loginWithUsername", token)
}
