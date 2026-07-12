package user

// PasswordHasher 是密码加密/校验的端口（接口）。
// 领域层只定义契约，具体算法（如 bcrypt）由基础设施层提供实现。
type PasswordHasher interface {
	// Hash 对明文密码加密，返回哈希串
	Hash(plain string) (string, error)
	// Compare 校验明文密码与哈希是否匹配
	Compare(hashed, plain string) error
}
