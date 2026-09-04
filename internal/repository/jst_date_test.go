package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL が未設定のためスキップ")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("DB 接続失敗: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// TestJstTodayExpr は jstToday 定数が DB 上で JST 日付を返すことを検証する。
func TestJstTodayExpr(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	var dbDate time.Time
	err := pool.QueryRow(ctx, "SELECT "+jstToday).Scan(&dbDate)
	if err != nil {
		t.Fatalf("jstToday クエリ失敗: %v", err)
	}

	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	expected := time.Now().In(jst).Format("2006-01-02")
	got := dbDate.Format("2006-01-02")
	if got != expected {
		t.Errorf("JST 日付が不一致: got=%s want=%s", got, expected)
	}
}

// TestJstTodayMatchesDailyTask は JST 日付で INSERT した daily_task が
// jstToday 式の WHERE で取得できることを検証する。
func TestJstTodayMatchesDailyTask(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	userID := setupTestUser(t, ctx, tx)
	questionID := setupTestQuestion(t, ctx, tx)

	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	todayJST := time.Now().In(jst).Format("2006-01-02")

	_, err = tx.Exec(ctx, `
		INSERT INTO daily_task (user_id, activity_date, slot_no, question_type, language, difficulty, question_id)
		VALUES ($1, $2, 1, 'code_reading', 'go', 3, $3)`,
		userID, todayJST, questionID)
	if err != nil {
		t.Fatalf("daily_task INSERT 失敗: %v", err)
	}

	var count int
	err = tx.QueryRow(ctx,
		`SELECT count(*) FROM daily_task WHERE user_id = $1 AND activity_date = `+jstToday,
		userID).Scan(&count)
	if err != nil {
		t.Fatalf("daily_task SELECT 失敗: %v", err)
	}
	if count != 1 {
		t.Errorf("JST 日付で INSERT した行が jstToday 式で取得できない: count=%d", count)
	}
}

// TestSrsStateDueOnUsesJST は srs_state の due_on が JST 基準で計算されることを検証する。
func TestSrsStateDueOnUsesJST(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	userID := setupTestUser(t, ctx, tx)
	questionID := setupTestQuestion(t, ctx, tx)

	_, err = tx.Exec(ctx, `
		INSERT INTO srs_state (user_id, question_id, due_on, last_reviewed_at)
		VALUES ($1, $2, `+jstToday+` + 1, now())`,
		userID, questionID)
	if err != nil {
		t.Fatalf("srs_state INSERT 失敗: %v", err)
	}

	var dueOn time.Time
	err = tx.QueryRow(ctx,
		`SELECT due_on FROM srs_state WHERE user_id = $1 AND question_id = $2`,
		userID, questionID).Scan(&dueOn)
	if err != nil {
		t.Fatalf("srs_state SELECT 失敗: %v", err)
	}

	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	expected := time.Now().In(jst).AddDate(0, 0, 1).Format("2006-01-02")
	got := dueOn.Format("2006-01-02")
	if got != expected {
		t.Errorf("due_on が JST+1 日と不一致: got=%s want=%s", got, expected)
	}
}

func setupTestUser(t *testing.T, ctx context.Context, tx pgx.Tx) string {
	t.Helper()
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO app_user (external_id, display_name)
		VALUES ('test-jst-' || gen_random_uuid(), 'JST Test User')
		RETURNING id`).Scan(&id)
	if err != nil {
		t.Fatalf("テストユーザー作成失敗: %v", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO user_progress (user_id) VALUES ($1)`, id)
	if err != nil {
		t.Fatalf("user_progress 作成失敗: %v", err)
	}
	return id
}

func setupTestQuestion(t *testing.T, ctx context.Context, tx pgx.Tx) string {
	t.Helper()
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO question (raw_source_id, type, status, difficulty, title, body, code_language,
		                      choices, correct_keys)
		VALUES ('00000000-0000-0000-0000-000000000001', 'code_reading', 'published', 3,
		        'JST test', 'body', 'go', '[{"key":"a","text":"A"}]', '["a"]')
		RETURNING id`).Scan(&id)
	if err != nil {
		t.Fatalf("テスト問題作成失敗: %v", err)
	}
	return id
}
