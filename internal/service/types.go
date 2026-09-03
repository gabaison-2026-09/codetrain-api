package service

import "github.com/gabaison-2026-09/codetrain-core/pkg/domain"

// UserWithProgress は /v1/me が返す内容。
type UserWithProgress struct {
	User     domain.User     `json:"user"`
	Progress domain.Progress `json:"progress"`
}

// CreateUserInput は POST /v1/me の作成内容。
// DisplayName は必須。AvatarURL は任意（未指定なら nil）。
type CreateUserInput struct {
	DisplayName string
	AvatarURL   *string
}

// QuestionSearchParams は GET /v1/questions のクエリ。handler が型変換・検証したあと
// service に渡す。Cursor は未デコードの文字列（空なら先頭頁）。
type QuestionSearchParams struct {
	SkillNodeID    string
	Type           domain.QuestionType
	Language       string
	Difficulty     *int
	Tags           []string
	Q              string
	UnansweredOnly bool
	Cursor         string
	Limit          int
}

// QuestionList は GET /v1/questions のレスポンス。questions は空でも []。
type QuestionList struct {
	Questions  []domain.QuestionSummary `json:"questions"`
	NextCursor *string                  `json:"next_cursor"`
}
