package service

import "errors"

// ErrUserNotFound は指定された external_id のユーザーが存在しないことを表す。
// repository.ErrNotFound をユースケースの語彙に翻訳したもので、
// handler はこれだけを見て 404 を返す（repository のエラーを直接見ない）。
var ErrUserNotFound = errors.New("ユーザーが見つかりません")
