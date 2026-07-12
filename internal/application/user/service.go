package userapp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wsc-zz/service/internal/domain/user"
)

// TokenIssuer 是令牌签发的端口（应用层定义契约，基础设施层提供 JWT 实现）。
type TokenIssuer interface {
	// Issue 为指定用户签发 token
	Issue(userID uint, username string) (string, error)
}

// Service 是用户应用服务，编排注册/登录等用例。
// 它只依赖领域接口与端口，不接触 GORM/bcrypt/JWT 等具体实现，
// 具体实现由 main.go 在组合根注入。
type Service struct {
	repo        user.UserRepository
	hasher      user.PasswordHasher
	tokenIssuer TokenIssuer
}

// NewService 构造应用服务，注入仓储、密码哈希器、令牌签发器。
func NewService(repo user.UserRepository, hasher user.PasswordHasher, tokenIssuer TokenIssuer) *Service {
	return &Service{repo: repo, hasher: hasher, tokenIssuer: tokenIssuer}
}

// Register 注册新用户，成功返回创建出的用户视图（不含密码）。
func (s *Service) Register(ctx context.Context, in RegisterInput) (*UserDTO, error) {
	// 用户名已存在则报错；其他底层错误直接向上抛
	existing, err := s.repo.FindByUsername(ctx, in.Username)
	if err != nil && !errors.Is(err, user.ErrUserNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, user.ErrUserAlreadyExists
	}

	hashed, err := s.hasher.Hash(in.Password)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	u := user.NewUser(in.Username, hashed, in.Phone, in.Nickname)
	if err := s.repo.Save(ctx, u); err != nil {
		return nil, err
	}
	dto := toUserDTO(u)
	return &dto, nil
}

// Login 用户名密码登录，成功返回 token 与用户信息。
func (s *Service) Login(ctx context.Context, in LoginInput) (*LoginResult, error) {
	u, err := s.repo.FindByUsername(ctx, in.Username)
	if err != nil {
		// 不暴露用户是否存在，统一返回凭证错误，避免用户枚举
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, user.ErrInvalidCredentials
		}
		return nil, err
	}

	if !u.VerifyPassword(in.Password, s.hasher) {
		return nil, user.ErrInvalidCredentials
	}
	if u.IsDisabled() {
		return nil, user.ErrUserDisabled
	}

	u.RecordLogin(in.IP, time.Now())
	if err := s.repo.Save(ctx, u); err != nil {
		return nil, fmt.Errorf("更新登录信息失败: %w", err)
	}

	token, err := s.tokenIssuer.Issue(u.UserID, u.Username)
	if err != nil {
		return nil, fmt.Errorf("生成 token 失败: %w", err)
	}

	return &LoginResult{
		Token: token,
		User:  toUserDTO(u),
	}, nil
}

// toUserDTO 将领域实体转为对外输出 DTO，避免泄漏领域实体结构与密码字段。
func toUserDTO(u *user.User) UserDTO {
	return UserDTO{
		UserID:       u.UserID,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
		Username:     u.Username,
		LoginChannel: u.LoginChannel,
		UnionID:      u.UnionID,
		OpenID:       u.OpenID,
		AlipayUID:    u.AlipayUID,
		Nickname:     u.Nickname,
		Phone:        u.Phone,
		Email:        u.Email,
		Avatar:       u.Avatar,
		Gender:       u.Gender,
		Birthday:     u.Birthday,
		Status:       u.Status,
		LastLoginIP:  u.LastLoginIP,
		LastLoginAt:  u.LastLoginAt,
	}
}
