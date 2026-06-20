// Package response 统一 HTTP 响应封装：{code, msg, data}（net/http 版）。
package response

import (
	"encoding/json"
	"net/http"

	"vibe-studio/backend/pkg/errorx"
)

type Body struct {
	Code int32  `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

// OK 成功响应（code=0，HTTP 200）。
func OK(w http.ResponseWriter, data any) {
	WriteJSON(w, http.StatusOK, Body{Code: 0, Msg: "ok", Data: data})
}

// Fail 失败响应：把任意 error 归一成业务错误，按其 HTTP 状态 + 业务码返回。
func Fail(w http.ResponseWriter, err error) {
	e := errorx.FromError(err)
	WriteJSON(w, e.HTTP, Body{Code: e.Code, Msg: e.Message})
}

// WriteJSON 写出任意 JSON 响应（供需要自定义状态码的场景，如就绪探针）。
func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
