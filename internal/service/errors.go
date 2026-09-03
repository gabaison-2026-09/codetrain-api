package service

import "errors"

// ErrUserNotFound は指定された external_id のユーザーが存在しないことを表す。
// 認証ユーザー解決ヘルパ（userResolver）が repository.ErrNotFound を
// ユースケースの語彙に翻訳したもので、handler はこれだけを見て 404 を返す
// （repository のエラーを直接見ない）。
var ErrUserNotFound = errors.New("ユーザーが見つかりません")

// ErrUserAlreadyProvisioned は同じ external_id の app_user が既に存在することを表す。
// repository.ErrAlreadyExists をユースケース語彙に翻訳したもので、
// handler はこれだけを見て 409 USER_ALREADY_PROVISIONED を返す。
var ErrUserAlreadyProvisioned = errors.New("ユーザーは既に作成済みです")
