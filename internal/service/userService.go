package service

import (
	"demo/global"
	"demo/internal/model"
	"demo/pkg/jwt"
	"demo/pkg/utils"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type UserService struct {
}

func (s *UserService) CreateUser(req model.CreateUserReq) error {
	_, err := s.GetUserByUsername(req.Username)
	if err == nil {
		return fmt.Errorf("用户已存在")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	user := model.User{
		Username: req.Username,
		Password: hashedPassword,
		Phone:    req.Phone,
	}
	return global.DB.Create(&user).Error
}

func (s *UserService) GetUserByUsername(username string) (model.User, error) {
	var user model.User
	err := global.DB.Where("username = ?", username).First(&user).Error
	return user, err
}

func (s *UserService) updateUser(user *model.User) error {
	return global.DB.Save(user).Error
}

func (s *UserService) LoginWithUsername(req model.LoginUserWithUsernameReq) (token string, err error) {
	user, err := s.GetUserByUsername(req.Username)
	if err != nil {
		return "", err
	}

	if utils.CheckPassword(req.Password, user.Password) != nil {
		return "", fmt.Errorf("密码错误")
	}
	 return jwt.GenerateToken(fmt.Sprintf("%d", user.UserID), user.Username)

}
