// Package repository は PostgreSQL へのアクセスをまとめる。
//
// スキーマの定義は codetrain-core（migrations）が持つ。api はそれを読むだけで、
// マイグレーションは打たない（REPOSITORIES.md §2.1）。
//
// この層の責務は **行の取得と型への詰め替えだけ**。
// 取得した行をどう組み立てるか・どう解釈するかは service 層が持つ。
// 依存の向きは handler → service → repository の一方向で、逆流させない。
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound は対象のレコードが無いことを表す。
// service 層がこれを受けて、ユースケースごとのエラーに翻訳する。
var ErrNotFound = errors.New("not found")

// ErrAlreadyExists は UNIQUE 制約違反など、既に同一キーの行があることを表す。
// service 層がこれを受けて、ユースケースごとのエラーに翻訳する。
var ErrAlreadyExists = errors.New("already exists")

// Postgres は pgx のコネクションプールを保持する。
type Postgres struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("DB プールの作成に失敗: %w", err)
	}
	return &Postgres{pool: pool}, nil
}

func (p *Postgres) Close() { p.pool.Close() }

// Ping は /healthz の db 判定に使う。
func (p *Postgres) Ping(ctx context.Context) error {
	var one int
	return p.pool.QueryRow(ctx, "SELECT 1").Scan(&one)
}
