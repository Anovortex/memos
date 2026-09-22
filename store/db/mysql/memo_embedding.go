package mysql

import (
	"context"
	"strings"

	"github.com/usememos/memos/store"
)

func (d *DB) UpsertMemoEmbedding(ctx context.Context, upsert *store.MemoEmbedding) (*store.MemoEmbedding, error) {
	stmt := "INSERT INTO `memo_embedding` (`memo_id`, `creator_id`, `model`, `vector`, `updated_ts`) VALUES (?, ?, ?, ?, ?) " +
		"ON DUPLICATE KEY UPDATE `creator_id` = VALUES(`creator_id`), `model` = VALUES(`model`), `vector` = VALUES(`vector`), `updated_ts` = VALUES(`updated_ts`)"
	if _, err := d.db.ExecContext(ctx, stmt, upsert.MemoID, upsert.CreatorID, upsert.Model, upsert.Vector, upsert.UpdatedTs); err != nil {
		return nil, err
	}
	return upsert, nil
}

func (d *DB) ListMemoEmbeddings(ctx context.Context, find *store.FindMemoEmbedding) ([]*store.MemoEmbedding, error) {
	where, args := []string{"1 = 1"}, []any{}

	if find.MemoID != nil {
		where, args = append(where, "`memo_id` = ?"), append(args, *find.MemoID)
	}
	if find.CreatorID != nil {
		where, args = append(where, "`creator_id` = ?"), append(args, *find.CreatorID)
	}
	if find.Model != nil {
		where, args = append(where, "`model` = ?"), append(args, *find.Model)
	}

	rows, err := d.db.QueryContext(ctx, "SELECT `memo_id`, `creator_id`, `model`, `vector`, `updated_ts` FROM `memo_embedding` WHERE "+strings.Join(where, " AND ")+" ORDER BY `memo_id` ASC", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []*store.MemoEmbedding{}
	for rows.Next() {
		me := &store.MemoEmbedding{}
		if err := rows.Scan(
			&me.MemoID,
			&me.CreatorID,
			&me.Model,
			&me.Vector,
			&me.UpdatedTs,
		); err != nil {
			return nil, err
		}
		list = append(list, me)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (d *DB) DeleteMemoEmbedding(ctx context.Context, delete *store.DeleteMemoEmbedding) error {
	_, err := d.db.ExecContext(ctx, "DELETE FROM `memo_embedding` WHERE `memo_id` = ?", delete.MemoID)
	return err
}

func (d *DB) ListMemosNeedingEmbedding(ctx context.Context, model string, limit int) ([]*store.Memo, error) {
	rows, err := d.db.QueryContext(ctx, "SELECT m.`id`, m.`creator_id`, m.`content` FROM `memo` m LEFT JOIN `memo_embedding` e ON e.`memo_id` = m.`id` AND e.`model` = ? WHERE e.`memo_id` IS NULL AND m.`row_status` = 'NORMAL' ORDER BY m.`id` ASC LIMIT ?", model, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []*store.Memo{}
	for rows.Next() {
		memo := &store.Memo{}
		if err := rows.Scan(
			&memo.ID,
			&memo.CreatorID,
			&memo.Content,
		); err != nil {
			return nil, err
		}
		list = append(list, memo)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}
