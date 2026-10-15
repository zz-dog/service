package addressapp

import (
	"context"
	"errors"

	"github.com/wsc-zz/service/internal/domain/address"
)

// TxRunner 是事务端口（应用层定义契约，infrastructure/persistence/transaction 提供）。
type TxRunner interface {
	// RunAddress 在事务内执行 fn，fn 收到与事务绑定的地址仓储
	RunAddress(ctx context.Context, fn func(address.AddressRepository) error) error
}

// Service 是收货地址应用服务，编排地址相关用例。
type Service struct {
	repo address.AddressRepository
	tx   TxRunner
}

// NewService 构造应用服务，注入地址仓储与事务端口。
func NewService(repo address.AddressRepository, tx TxRunner) *Service {
	return &Service{repo: repo, tx: tx}
}

// Create 新增收货地址。
func (s *Service) Create(ctx context.Context, in CreateInput) (*AddressDto, error) {
	count, err := s.repo.CountByUserID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	if count >= address.AddressMaxCount {
		return nil, address.ErrAddressLimit
	}

	a, err := address.NewAddress(in.UserID, in.Receiver, in.Phone, in.Province, in.City, in.District, in.Detail)
	if err != nil {
		return nil, err
	}

	// 第一条地址自动设为默认；显式 isDefault=true 同样需要清掉其它默认标记
	makeDefault := count == 0 || in.IsDefault
	if makeDefault {
		a.IsDefault = true
		if count > 0 {
			return nil, s.saveAsDefault(ctx, a)
		}
	}
	if err := s.repo.Save(ctx, a); err != nil {
		return nil, err
	}
	dto := toDto(a)
	return &dto, nil
}

// saveAsDefault 在同一事务内清零其它默认标记并保存默认地址。
func (s *Service) saveAsDefault(ctx context.Context, a *address.Address) error {
	return s.tx.RunAddress(ctx, func(txRepo address.AddressRepository) error {
		if err := txRepo.ClearDefault(ctx, a.UserID); err != nil {
			return err
		}
		return txRepo.Save(ctx, a)
	})
}

// List 查询当前用户地址列表，默认地址排最前。
func (s *Service) List(ctx context.Context, userID uint) ([]*AddressDto, error) {
	addrs, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	dtos := make([]*AddressDto, len(addrs))
	for i, a := range addrs {
		d := toDto(a)
		dtos[i] = &d
	}
	return dtos, nil
}

// getOwned 查询地址并做归属校验：地址不存在或不属于该用户都返回 ErrAddressNotFound，
// 不暴露他人地址的存在性。
func (s *Service) getOwned(ctx context.Context, userID, addressID uint) (*address.Address, error) {
	a, err := s.repo.FindByID(ctx, addressID)
	if err != nil {
		if errors.Is(err, address.ErrAddressNotFound) {
			return nil, address.ErrAddressNotFound
		}
		return nil, err
	}
	if !a.BelongsTo(userID) {
		return nil, address.ErrAddressNotFound
	}
	return a, nil
}

// Get 查询地址详情。
func (s *Service) Get(ctx context.Context, userID, addressID uint) (*AddressDto, error) {
	a, err := s.getOwned(ctx, userID, addressID)
	if err != nil {
		return nil, err
	}
	dto := toDto(a)
	return &dto, nil
}

// Update 部分更新地址，未传字段不修改。
func (s *Service) Update(ctx context.Context, in UpdateInput) (*AddressDto, error) {
	a, err := s.getOwned(ctx, in.UserID, in.AddressID)
	if err != nil {
		return nil, err
	}
	if err := a.Update(in.Receiver, in.Phone, in.Province, in.City, in.District, in.Detail); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, a); err != nil {
		return nil, err
	}
	dto := toDto(a)
	return &dto, nil
}

// Delete 软删除地址。
func (s *Service) Delete(ctx context.Context, userID, addressID uint) error {
	a, err := s.getOwned(ctx, userID, addressID)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, a)
}

// SetDefault 把指定地址设为默认：事务内先清零其它默认标记，再置该条为默认。
func (s *Service) SetDefault(ctx context.Context, userID, addressID uint) error {
	a, err := s.getOwned(ctx, userID, addressID)
	if err != nil {
		return err
	}
	a.IsDefault = true
	return s.tx.RunAddress(ctx, func(txRepo address.AddressRepository) error {
		if err := txRepo.ClearDefault(ctx, a.UserID); err != nil {
			return err
		}
		return txRepo.Save(ctx, a)
	})
}

// toDto 将领域实体转为对外输出 DTO。
func toDto(a *address.Address) AddressDto {
	return AddressDto{
		AddressID: a.AddressID,
		Receiver:  a.Receiver,
		Phone:     a.Phone,
		Province:  a.Province,
		City:      a.City,
		District:  a.District,
		Detail:    a.Detail,
		IsDefault: a.IsDefault,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
}
