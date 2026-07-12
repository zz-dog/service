package model

type RegisterUserReq struct {
	Username string `json:"username" binding:"required,min=2,max=10"`
	Password string `json:"password" binding:"required,min=6,max=20"`
	Phone    string `json:"phone" binding:"len=11"`
	Nickname string `json:"nickname" binding:"min=2,max=10"`
}

type LoginUserWithUsernameReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Ip       string `json:"ip"`
}

type LoginUserResp struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
