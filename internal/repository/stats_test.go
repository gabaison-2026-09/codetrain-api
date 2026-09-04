package repository

import (
	"context"
	"testing"
)

// TestListTypeStats は user_type_stat の行を種別×言語順に、
// language='' や last_difficulty=NULL を含めて返すことを検証する。
// ListTypeStats は pool を直接使うため、コミット済みデータで検証し後始末する。
func TestListTypeStats(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	p := &Postgres{pool: pool}

	var userID string
	err := pool.QueryRow(ctx, `
		INSERT INTO app_user (external_id, display_name)
		VALUES ('test-stats-' || gen_random_uuid(), 'Stats Test User')
		RETURNING id`).Scan(&userID)
	if err != nil {
		t.Fatalf("テストユーザー作成失敗: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM app_user WHERE id = $1`, userID)
	})

	_, err = pool.Exec(ctx, `
		INSERT INTO user_type_stat (user_id, question_type, language, attempts, corrects, last_difficulty)
		VALUES ($1, 'code_reading', 'typescript', 42, 35, 3),
		       ($1, 'output_prediction', '', 10, 6, NULL)`, userID)
	if err != nil {
		t.Fatalf("user_type_stat INSERT 失敗: %v", err)
	}

	got, err := p.ListTypeStats(ctx, userID)
	if err != nil {
		t.Fatalf("ListTypeStats: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("stats = %+v, want 2 件", got)
	}
	// ORDER BY question_type, language: code_reading が先
	if got[0].QuestionType != "code_reading" || got[0].Language != "typescript" ||
		got[0].Attempts != 42 || got[0].Corrects != 35 ||
		got[0].LastDifficulty == nil || *got[0].LastDifficulty != 3 {
		t.Errorf("got[0] = %+v", got[0])
	}
	if got[1].QuestionType != "output_prediction" || got[1].Language != "" || got[1].LastDifficulty != nil {
		t.Errorf("got[1] = %+v, want language=\"\" last_difficulty=NULL", got[1])
	}
	// Accuracy は repository では埋めない
	if got[0].Accuracy != 0 {
		t.Errorf("repository が Accuracy を埋めている: %v", got[0].Accuracy)
	}
}

// TestListTypeStatsEmpty は行が無いユーザーで空スライスを返すことを検証する。
func TestListTypeStatsEmpty(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	p := &Postgres{pool: pool}

	var userID string
	err := pool.QueryRow(ctx, `
		INSERT INTO app_user (external_id, display_name)
		VALUES ('test-stats-' || gen_random_uuid(), 'Stats Test User')
		RETURNING id`).Scan(&userID)
	if err != nil {
		t.Fatalf("テストユーザー作成失敗: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM app_user WHERE id = $1`, userID)
	})

	got, err := p.ListTypeStats(ctx, userID)
	if err != nil {
		t.Fatalf("ListTypeStats: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("stats = %#v, want 空スライス", got)
	}
}
