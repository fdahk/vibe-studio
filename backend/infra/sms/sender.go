// Package sms 短信发送与验证码存储的基础设施实现。
// Sender 抽象发送通道：以后从 console 切到火山引擎/阿里云等真实网关时，只换实现、业务不动（易扩展）。
package sms

import (
	"context"
	"log/slog"
)

// Sender 短信发送端口。实现：ConsoleSender(dev) / 未来 VolcengineSender、AliyunSender 等。
type Sender interface {
	Send(ctx context.Context, phone, code string) error
}

// ConsoleSender 开发用发送器：把验证码打到日志，不真正下发短信。
// 生产替换为真实网关实现即可（只换这一处）。
type ConsoleSender struct{}

func NewConsoleSender() *ConsoleSender { return &ConsoleSender{} }

func (ConsoleSender) Send(_ context.Context, phone, code string) error {
	slog.Info("【DEV 短信】验证码已生成（未真正下发，生产请接入真实短信网关）", "phone", phone, "code", code)
	return nil
}
