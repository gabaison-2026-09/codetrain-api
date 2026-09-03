package service

import (
	"context"
	"errors"
	"testing"

	"github.com/gabaison-2026-09/codetrain-core/pkg/domain"

	"github.com/gabaison-2026-09/codetrain-api/internal/repository"
)

type fakeTaskOptionRepo struct {
	find   func(context.Context, string) (domain.User, domain.Progress, error)
	list   func(context.Context) ([]domain.TaskOption, error)
	exists func(context.Context, domain.QuestionType, string, *int) (bool, error)
	upsert func(context.Context, string, domain.TaskConfig) (domain.TaskConfig, error)
}

func (f fakeTaskOptionRepo) FindUserByExternalID(ctx context.Context, externalID string) (domain.User, domain.Progress, error) {
	return f.find(ctx, externalID)
}

func (f fakeTaskOptionRepo) ListTaskOptions(ctx context.Context) ([]domain.TaskOption, error) {
	return f.list(ctx)
}

func (f fakeTaskOptionRepo) OptionExists(ctx context.Context, questionType domain.QuestionType, language string, difficulty *int) (bool, error) {
	return f.exists(ctx, questionType, language, difficulty)
}

func (f fakeTaskOptionRepo) UpsertUserTask(ctx context.Context, userID string, slot domain.TaskConfig) (domain.TaskConfig, error) {
	return f.upsert(ctx, userID, slot)
}

func TestTaskSlotList(t *testing.T) {
	t.Run("ユーザー確認後に候補を返す", func(t *testing.T) {
		var gotExternalID string
		repo := fakeTaskOptionRepo{
			find: func(_ context.Context, externalID string) (domain.User, domain.Progress, error) {
				gotExternalID = externalID
				return domain.User{ID: "u1"}, domain.Progress{}, nil
			},
			list: func(context.Context) ([]domain.TaskOption, error) {
				return []domain.TaskOption{{QuestionType: domain.QuestionTypeCodeReading, Language: "typescript", Difficulty: 1}}, nil
			},
		}

		got, err := NewTaskOptions(repo).List(context.Background(), "seed-user-01")
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if gotExternalID != "seed-user-01" {
			t.Errorf("external_id = %q, want seed-user-01", gotExternalID)
		}
		if len(got) != 1 || got[0].Language != "typescript" {
			t.Errorf("options = %+v", got)
		}
	})

	t.Run("候補0件は nil ではなく空配列を返す", func(t *testing.T) {
		repo := fakeTaskOptionRepo{
			find: func(context.Context, string) (domain.User, domain.Progress, error) {
				return domain.User{ID: "u1"}, domain.Progress{}, nil
			},
			list: func(context.Context) ([]domain.TaskOption, error) { return nil, nil },
		}

		got, err := NewTaskOptions(repo).List(context.Background(), "seed-user-01")
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if got == nil || len(got) != 0 {
			t.Errorf("options = %#v, want non-nil empty slice", got)
		}
	})

	t.Run("未登録ユーザーは候補を取得せず ErrUserNotFound", func(t *testing.T) {
		listCalled := false
		repo := fakeTaskOptionRepo{
			find: func(context.Context, string) (domain.User, domain.Progress, error) {
				return domain.User{}, domain.Progress{}, repository.ErrNotFound
			},
			list: func(context.Context) ([]domain.TaskOption, error) {
				listCalled = true
				return nil, nil
			},
		}

		_, err := NewTaskOptions(repo).List(context.Background(), "no-such-user")
		if !errors.Is(err, ErrUserNotFound) {
			t.Errorf("err = %v, want %v", err, ErrUserNotFound)
		}
		if listCalled {
			t.Error("ユーザー確認失敗後に ListTaskOptions が呼ばれた")
		}
	})

	t.Run("候補取得エラーを伝播する", func(t *testing.T) {
		wantErr := errors.New("DB 障害")
		repo := fakeTaskOptionRepo{
			find: func(context.Context, string) (domain.User, domain.Progress, error) {
				return domain.User{ID: "u1"}, domain.Progress{}, nil
			},
			list: func(context.Context) ([]domain.TaskOption, error) { return nil, wantErr },
		}

		_, err := NewTaskOptions(repo).List(context.Background(), "seed-user-01")
		if !errors.Is(err, wantErr) {
			t.Errorf("err = %v, want %v", err, wantErr)
		}
	})
}

func TestTaskSlotSetSlot(t *testing.T) {
	difficulty := 3
	want := domain.TaskConfig{
		SlotNo:       2,
		QuestionType: domain.QuestionTypeBugFinding,
		Language:     "typescript",
		Difficulty:   &difficulty,
	}

	t.Run("ユーザーと候補を確認してスロットを保存する", func(t *testing.T) {
		repo := fakeTaskOptionRepo{
			find: func(_ context.Context, externalID string) (domain.User, domain.Progress, error) {
				if externalID != "seed-user-01" {
					t.Errorf("externalID = %q", externalID)
				}
				return domain.User{ID: "u1"}, domain.Progress{}, nil
			},
			exists: func(_ context.Context, questionType domain.QuestionType, language string, gotDifficulty *int) (bool, error) {
				if questionType != want.QuestionType || language != want.Language || gotDifficulty == nil || *gotDifficulty != difficulty {
					t.Errorf("option = (%q, %q, %v)", questionType, language, gotDifficulty)
				}
				return true, nil
			},
			upsert: func(_ context.Context, userID string, slot domain.TaskConfig) (domain.TaskConfig, error) {
				if userID != "u1" || slot.SlotNo != want.SlotNo {
					t.Errorf("userID = %q, slot = %+v", userID, slot)
				}
				return slot, nil
			},
		}

		got, err := NewTaskOptions(repo).SetSlot(context.Background(), "seed-user-01", want)
		if err != nil {
			t.Fatalf("SetSlot: %v", err)
		}
		if got.SlotNo != want.SlotNo || got.QuestionType != want.QuestionType || got.Language != want.Language {
			t.Errorf("slot = %+v, want %+v", got, want)
		}
	})

	t.Run("difficulty nil をそのまま候補確認に渡す", func(t *testing.T) {
		slot := want
		slot.Difficulty = nil
		repo := fakeTaskOptionRepo{
			find: func(context.Context, string) (domain.User, domain.Progress, error) {
				return domain.User{ID: "u1"}, domain.Progress{}, nil
			},
			exists: func(_ context.Context, _ domain.QuestionType, _ string, difficulty *int) (bool, error) {
				if difficulty != nil {
					t.Errorf("difficulty = %v, want nil", difficulty)
				}
				return true, nil
			},
			upsert: func(_ context.Context, _ string, slot domain.TaskConfig) (domain.TaskConfig, error) {
				return slot, nil
			},
		}

		if _, err := NewTaskOptions(repo).SetSlot(context.Background(), "seed-user-01", slot); err != nil {
			t.Fatalf("SetSlot: %v", err)
		}
	})

	t.Run("スロット番号が範囲外ならリポジトリを呼ばない", func(t *testing.T) {
		repo := fakeTaskOptionRepo{}
		_, err := NewTaskOptions(repo).SetSlot(context.Background(), "seed-user-01", domain.TaskConfig{SlotNo: 6})
		if !errors.Is(err, ErrTaskSlotNoInvalid) {
			t.Errorf("err = %v, want %v", err, ErrTaskSlotNoInvalid)
		}
	})

	t.Run("未登録ユーザーは候補確認しない", func(t *testing.T) {
		repo := fakeTaskOptionRepo{
			find: func(context.Context, string) (domain.User, domain.Progress, error) {
				return domain.User{}, domain.Progress{}, repository.ErrNotFound
			},
			exists: func(context.Context, domain.QuestionType, string, *int) (bool, error) {
				t.Fatal("OptionExists が呼ばれた")
				return false, nil
			},
		}
		_, err := NewTaskOptions(repo).SetSlot(context.Background(), "no-such-user", want)
		if !errors.Is(err, ErrUserNotFound) {
			t.Errorf("err = %v, want %v", err, ErrUserNotFound)
		}
	})

	t.Run("候補に無い組み合わせは保存しない", func(t *testing.T) {
		repo := fakeTaskOptionRepo{
			find: func(context.Context, string) (domain.User, domain.Progress, error) {
				return domain.User{ID: "u1"}, domain.Progress{}, nil
			},
			exists: func(context.Context, domain.QuestionType, string, *int) (bool, error) {
				return false, nil
			},
			upsert: func(context.Context, string, domain.TaskConfig) (domain.TaskConfig, error) {
				t.Fatal("UpsertUserTask が呼ばれた")
				return domain.TaskConfig{}, nil
			},
		}
		_, err := NewTaskOptions(repo).SetSlot(context.Background(), "seed-user-01", want)
		if !errors.Is(err, ErrTaskSlotOptionInvalid) {
			t.Errorf("err = %v, want %v", err, ErrTaskSlotOptionInvalid)
		}
	})

	t.Run("候補確認と保存のエラーを伝播する", func(t *testing.T) {
		for _, tc := range []struct {
			name      string
			existsErr error
			upsertErr error
		}{
			{name: "候補確認", existsErr: errors.New("option DB 障害")},
			{name: "保存", upsertErr: errors.New("upsert DB 障害")},
		} {
			t.Run(tc.name, func(t *testing.T) {
				repo := fakeTaskOptionRepo{
					find: func(context.Context, string) (domain.User, domain.Progress, error) {
						return domain.User{ID: "u1"}, domain.Progress{}, nil
					},
					exists: func(context.Context, domain.QuestionType, string, *int) (bool, error) {
						return tc.existsErr == nil, tc.existsErr
					},
					upsert: func(context.Context, string, domain.TaskConfig) (domain.TaskConfig, error) {
						return domain.TaskConfig{}, tc.upsertErr
					},
				}
				_, err := NewTaskOptions(repo).SetSlot(context.Background(), "seed-user-01", want)
				wantErr := tc.existsErr
				if wantErr == nil {
					wantErr = tc.upsertErr
				}
				if !errors.Is(err, wantErr) {
					t.Errorf("err = %v, want %v", err, wantErr)
				}
			})
		}
	})
}

type fakeTaskSlotRepo struct {
	list   func(context.Context, string) ([]domain.TaskConfig, error)
	delete func(context.Context, string, int) error
}

func (f fakeTaskSlotRepo) ListUserTasks(ctx context.Context, userID string) ([]domain.TaskConfig, error) {
	return f.list(ctx, userID)
}

func (f fakeTaskSlotRepo) DeleteUserTask(ctx context.Context, userID string, slotNo int) error {
	return f.delete(ctx, userID, slotNo)
}

func TestTaskSlotListSlots(t *testing.T) {
	var gotUserID string
	difficulty := 3
	svc := NewTaskSlot(fakeUserRepo{
		find: func(context.Context, string) (domain.User, domain.Progress, error) {
			return domain.User{ID: "u1"}, domain.Progress{}, nil
		},
	}, fakeTaskSlotRepo{
		list: func(_ context.Context, userID string) ([]domain.TaskConfig, error) {
			gotUserID = userID
			return []domain.TaskConfig{
				{SlotNo: 1, QuestionType: domain.QuestionTypeCodeReading, Language: "typescript"},
				{SlotNo: 2, QuestionType: domain.QuestionTypeOutputPrediction, Difficulty: &difficulty},
			}, nil
		},
	})

	got, err := svc.ListSlots(context.Background(), "seed-user-01")
	if err != nil {
		t.Fatalf("ListSlots: %v", err)
	}
	if gotUserID != "u1" {
		t.Fatalf("repository userID = %q, want %q", gotUserID, "u1")
	}
	if len(got) != 2 || got[1].Difficulty == nil || *got[1].Difficulty != 3 {
		t.Fatalf("slots = %+v", got)
	}
}

func TestTaskSlotListSlotsEmptyIsArray(t *testing.T) {
	svc := NewTaskSlot(fakeUserRepo{
		find: func(context.Context, string) (domain.User, domain.Progress, error) {
			return domain.User{ID: "u1"}, domain.Progress{}, nil
		},
	}, fakeTaskSlotRepo{
		list: func(context.Context, string) ([]domain.TaskConfig, error) { return nil, nil },
	})

	got, err := svc.ListSlots(context.Background(), "seed-user-01")
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("slots = %#v, err = %v", got, err)
	}
}

func TestTaskSlotListSlotsResolvesUserError(t *testing.T) {
	svc := NewTaskSlot(fakeUserRepo{
		find: func(context.Context, string) (domain.User, domain.Progress, error) {
			return domain.User{}, domain.Progress{}, errors.New("DB障害")
		},
	}, fakeTaskSlotRepo{
		list: func(context.Context, string) ([]domain.TaskConfig, error) { t.Fatal("not called"); return nil, nil },
	})

	if _, err := svc.ListSlots(context.Background(), "seed-user-01"); err == nil {
		t.Fatal("expected error")
	}
}

func TestTaskSlotDeleteSlot(t *testing.T) {
	t.Run("ユーザーを解決して指定スロットを削除する", func(t *testing.T) {
		var gotUserID string
		var gotSlotNo int
		svc := NewTaskSlot(fakeUserRepo{
			find: func(_ context.Context, externalID string) (domain.User, domain.Progress, error) {
				if externalID != "seed-user-01" {
					t.Errorf("externalID = %q", externalID)
				}
				return domain.User{ID: "u1"}, domain.Progress{}, nil
			},
		}, fakeTaskSlotRepo{
			delete: func(_ context.Context, userID string, slotNo int) error {
				gotUserID = userID
				gotSlotNo = slotNo
				return nil
			},
		})

		if err := svc.DeleteSlot(context.Background(), "seed-user-01", 3); err != nil {
			t.Fatalf("DeleteSlot: %v", err)
		}
		if gotUserID != "u1" || gotSlotNo != 3 {
			t.Errorf("DeleteUserTask(%q, %d), want (%q, %d)", gotUserID, gotSlotNo, "u1", 3)
		}
	})

	t.Run("スロット番号が範囲外ならリポジトリを呼ばない", func(t *testing.T) {
		svc := NewTaskSlot(fakeUserRepo{}, fakeTaskSlotRepo{})
		for _, slotNo := range []int{0, 6} {
			if err := svc.DeleteSlot(context.Background(), "seed-user-01", slotNo); !errors.Is(err, ErrTaskSlotNoInvalid) {
				t.Errorf("slotNo = %d, err = %v, want %v", slotNo, err, ErrTaskSlotNoInvalid)
			}
		}
	})

	t.Run("未登録ユーザーなら削除しない", func(t *testing.T) {
		svc := NewTaskSlot(fakeUserRepo{
			find: func(context.Context, string) (domain.User, domain.Progress, error) {
				return domain.User{}, domain.Progress{}, repository.ErrNotFound
			},
		}, fakeTaskSlotRepo{
			delete: func(context.Context, string, int) error {
				t.Fatal("DeleteUserTask が呼ばれた")
				return nil
			},
		})

		if err := svc.DeleteSlot(context.Background(), "no-such-user", 3); !errors.Is(err, ErrUserNotFound) {
			t.Errorf("err = %v, want %v", err, ErrUserNotFound)
		}
	})

	t.Run("削除対象が無ければスロット番号エラーに翻訳する", func(t *testing.T) {
		svc := NewTaskSlot(fakeUserRepo{
			find: func(context.Context, string) (domain.User, domain.Progress, error) {
				return domain.User{ID: "u1"}, domain.Progress{}, nil
			},
		}, fakeTaskSlotRepo{
			delete: func(context.Context, string, int) error { return repository.ErrNotFound },
		})

		err := svc.DeleteSlot(context.Background(), "seed-user-01", 3)
		if !errors.Is(err, ErrTaskSlotNoInvalid) {
			t.Errorf("err = %v, want %v", err, ErrTaskSlotNoInvalid)
		}
		if errors.Is(err, repository.ErrNotFound) {
			t.Errorf("repository.ErrNotFound が service から漏れている: %v", err)
		}
	})

	t.Run("削除エラーを伝播する", func(t *testing.T) {
		wantErr := errors.New("DB 障害")
		svc := NewTaskSlot(fakeUserRepo{
			find: func(context.Context, string) (domain.User, domain.Progress, error) {
				return domain.User{ID: "u1"}, domain.Progress{}, nil
			},
		}, fakeTaskSlotRepo{
			delete: func(context.Context, string, int) error { return wantErr },
		})

		if err := svc.DeleteSlot(context.Background(), "seed-user-01", 3); !errors.Is(err, wantErr) {
			t.Errorf("err = %v, want %v", err, wantErr)
		}
	})
}
