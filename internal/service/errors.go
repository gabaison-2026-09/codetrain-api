package service

import "errors"

// ErrUserNotFound は指定された external_id のユーザーが存在しないことを表す。
// 認証ユーザー解決ヘルパ（userResolver）が repository.ErrNotFound を
// ユースケースの語彙に翻訳したもので、handler はこれだけを見て 404 を返す
// （repository のエラーを直接見ない）。
var ErrUserNotFound = errors.New("ユーザーが見つかりません")

// ErrQuestionNotFound は指定された問題が存在しない（または published でない）ことを表す。
// repository.ErrNotFound をユースケースの語彙に翻訳したもので、
// handler はこれだけを見て 404 QUESTION_NOT_FOUND を返す。
var ErrQuestionNotFound = errors.New("問題が見つかりません")

// ErrUserAlreadyProvisioned は同じ external_id の app_user が既に存在することを表す。
// repository.ErrAlreadyExists をユースケース語彙に翻訳したもので、
// handler はこれだけを見て 409 USER_ALREADY_PROVISIONED を返す。
var ErrUserAlreadyProvisioned = errors.New("ユーザーは既に作成済みです")

// ErrTaskSlotNoInvalid はタスクスロット番号が 1〜5 の範囲外であることを表す。
var ErrTaskSlotNoInvalid = errors.New("タスクスロット番号が不正です")

// ErrTaskSlotOptionInvalid は指定されたタスク条件が利用可能な候補に存在しないことを表す。
var ErrTaskSlotOptionInvalid = errors.New("タスク条件の組み合わせが不正です")
