package service

import "errors"

// ErrUserNotFound は指定された external_id のユーザーが存在しないことを表す。
// 認証ユーザー解決ヘルパ（userResolver）が repository.ErrNotFound を
// ユースケースの語彙に翻訳したもので、handler はこれだけを見て 404 を返す
// （repository のエラーを直接見ない）。
var ErrUserNotFound = errors.New("ユーザーが見つかりません")
