package model

type CreateUserReq struct {
	Username string `json:"username" binding:"required,min=2,max=10"`
	Password string `json:"password" binding:"required,min=6,max=20"`
	Phone    string `json:"phone" binding:"required,len=11"`
}

type LoginUserWithUsernameReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}
