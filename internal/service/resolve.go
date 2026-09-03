package service

import (
	"context"
	"errors"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"

	"github.com/gabaison-2026-09/codetrain-api/internal/repository"
)

// userResolver は認証済みユーザーの sub（external_id）から app_user 行を
// 解決する共通ロジック。認証必須の service はこれを内包して再利用する。
//
// repository.ErrNotFound を ErrUserNotFound に翻訳する唯一の場所であり、
// handler はこの service 語彙だけを見てステータス変換する。
type userResolver struct {
	repo UserLookupRepository
}

// resolveUser は sub からユーザーと進捗を返す。
// 該当ユーザーがいない場合は ErrUserNotFound を返す。
// それ以外のエラーはそのまま伝播する。
func (r userResolver) resolveUser(ctx context.Context, sub string) (domain.User, domain.Progress, error) {
	user, progress, err := r.repo.FindUserByExternalID(ctx, sub)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.User{}, domain.Progress{}, ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, domain.Progress{}, err
	}
	return user, progress, nil
}

// resolveUserID は進捗が不要な service 向けの軽量版。
// 内部の app_user.id（uuid）を返す。未登録なら ErrUserNotFound。
func (r userResolver) resolveUserID(ctx context.Context, sub string) (string, error) {
	user, _, err := r.resolveUser(ctx, sub)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}
