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

// AdminQuestionSearchParams は GET /v1/admin/questions のクエリ。
// Cursor は未デコードの文字列（空なら先頭頁）。
type AdminQuestionSearchParams struct {
	Status   domain.QuestionStatus
	Type     domain.QuestionType
	Language string
	SkillID  string
	Q        string
	Cursor   string
	Limit    int
}

// AdminQuestionList は管理者向け問題一覧のレスポンス。
type AdminQuestionList struct {
	Questions  []domain.AdminQuestionSummary `json:"questions"`
	NextCursor *string                       `json:"next_cursor"`
}

// ReviewQueueParams は GET /v1/admin/review-queue のクエリ。
type ReviewQueueParams struct {
	Cursor string
	Limit  int
}

// ReviewQueueList は未レビュー問題一覧のレスポンス。
type ReviewQueueList struct {
	Items      []domain.ReviewQueueItem `json:"items"`
	NextCursor *string                  `json:"next_cursor"`
}

// SubmitAttemptInput は回答送信の入力。DurationMS は未指定なら nil。
type SubmitAttemptInput struct {
	SelectedKeys []string
	DurationMS   *int
}
