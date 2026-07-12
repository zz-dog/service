package service

import (
	"demo/global"
	"demo/internal/model"
	"demo/pkg/jwt"
	"demo/pkg/utils"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type UserService struct {
}

func (s *UserService) RegisterUser(req model.RegisterUserReq) error {
	// 检查用户名是否已存在
	_, err := s.GetUserByUsername(req.Username)
	if err == nil {
		return fmt.Errorf("用户已存在")
	}

	// 判断是否是其他错误，而不是记录未找到的错误
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	newUser := model.User{
		Username: req.Username,
		Password: hashedPassword,
		Phone:    req.Phone,
		Nickname: req.Nickname,
	}
	return global.DB.Create(&newUser).Error
}

func (s *UserService) GetUserByUsername(username string) (model.User, error) {
	var user model.User
	err := global.DB.Where("username = ?", username).First(&user).Error
	return user, err
}

func (s *UserService) updateUser(user *model.User) error {
	return global.DB.Save(user).Error
}

func (s *UserService) LoginWithUsername(req model.LoginUserWithUsernameReq) (*model.LoginUserResp, error) {
	user, err := s.GetUserByUsername(req.Username)
	if err != nil {
		return nil, fmt.Errorf("用户不存在")
	}
	// 检查密码是否正确
	if utils.CheckPassword(req.Password, user.Password) != nil {
		return nil, fmt.Errorf("账号或密码错误")
	}
	// 检查账号状态
	if user.Status == 0 {
		return nil, fmt.Errorf("账号已被禁用")
	}
	// 更新最后登录时间和IP
	now := time.Now()
	user.LastLoginAt = &now
	user.LastLoginIP = req.Ip
	if err := s.updateUser(&user); err != nil {
		return nil, fmt.Errorf("更新用户登录信息失败: %w", err)
	}

	// 生成JWT token
	token, err := jwt.GenerateToken(fmt.Sprintf("%d", user.UserID), user.Username)
	if err != nil {
		return nil, fmt.Errorf("生成token失败: %w", err)
	}

	// 密码字段不返回
	user.Password = ""
	return &model.LoginUserResp{
		Token: token,
		User:  user,
	}, nil
}
