package response

/*
业务错误码体系
功能：统一定义 API 业务错误码，前端可根据错误码做精准处理。
格式：模块前缀(2位) + 错误序号(3位)
  - 10xxx: 认证相关
  - 20xxx: 用户相关
  - 30xxx: 隧道相关
  - 40xxx: 节点相关
  - 50xxx: 系统相关
  - 90xxx: 通用错误
*/

const (
	/* 认证相关 10xxx */
	ErrCodeAuthInvalidCredentials = 10001 /* 用户名或密码错误 */
	ErrCodeAuthTokenExpired       = 10002 /* 令牌已过期 */
	ErrCodeAuthTokenInvalid       = 10003 /* 令牌无效 */
	ErrCodeAuthAccountDisabled    = 10004 /* 账户已禁用 */
	ErrCodeAuthCaptchaRequired    = 10005 /* 需要验证码 */
	ErrCodeAuthCaptchaInvalid     = 10006 /* 验证码无效 */
	ErrCodeAuthPermissionDenied   = 10007 /* 权限不足 */

	/* 用户相关 20xxx */
	ErrCodeUserNotFound           = 20001 /* 用户不存在 */
	ErrCodeUserAlreadyExists      = 20002 /* 用户已存在 */
	ErrCodeUserEmailExists        = 20003 /* 邮箱已被使用 */
	ErrCodeUserPasswordWeak       = 20004 /* 密码强度不足 */
	ErrCodeUserPasswordMismatch   = 20005 /* 旧密码不正确 */

	/* 隧道相关 30xxx */
	ErrCodeTunnelNotFound         = 30001 /* 隧道不存在 */
	ErrCodeTunnelPortConflict     = 30002 /* 端口冲突 */
	ErrCodeTunnelLimitExceeded    = 30003 /* 隧道数量超限 */
	ErrCodeTunnelInvalidConfig    = 30004 /* 隧道配置无效 */

	/* 节点相关 40xxx */
	ErrCodeNodeNotFound           = 40001 /* 节点不存在 */
	ErrCodeNodeOffline            = 40002 /* 节点离线 */
	ErrCodeNodeCKInvalid          = 40003 /* 连接密钥无效 */
	ErrCodeNodeGroupNotFound      = 40004 /* 节点组不存在 */

	/* 系统相关 50xxx */
	ErrCodeSystemSettingNotFound  = 50001 /* 系统配置不存在 */
	ErrCodeSystemMaintenanceMode  = 50002 /* 系统维护中 */

	/* 通用错误 90xxx */
	ErrCodeBadRequest             = 90001 /* 请求参数无效 */
	ErrCodeRateLimited            = 90002 /* 请求过于频繁 */
	ErrCodeInternalError          = 90003 /* 服务器内部错误 */
	ErrCodeNotFound               = 90004 /* 资源不存在 */
	ErrCodeConflict               = 90005 /* 资源冲突 */
)
