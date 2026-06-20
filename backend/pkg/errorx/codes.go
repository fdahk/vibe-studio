package errorx

// 错误码约定：0=成功；1xxxx=通用；2xxxx=用户/鉴权；后续领域各占一段。
var (
	ErrBadRequest   = New(10001, "请求参数错误", 400)
	ErrUnauthorized = New(10002, "未登录或登录已过期", 401)
	ErrForbidden    = New(10003, "无权限", 403)
	ErrNotFound     = New(10004, "资源不存在", 404)
	ErrConflict     = New(10005, "资源已存在", 409)
	ErrInternal     = New(10500, "服务器内部错误", 500)
)
