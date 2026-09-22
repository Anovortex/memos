package mysql

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
)

func (d *DB) IncrementUserAIUsage(ctx context.Context, userID int32, usageDate string, tokens int64) error {
	stmt := "INSERT INTO `user_ai_usage` (`user_id`, `usage_date`, `tokens`) VALUES (?, ?, ?) " +
		"ON DUPLICATE KEY UPDATE `tokens` = `tokens` + VALUES(`tokens`)"
	_, err := d.db.ExecContext(ctx, stmt, userID, usageDate, tokens)
	return err
}

func (d *DB) GetUserAIUsage(ctx context.Context, userID int32, usageDate string) (int64, error) {
	var tokens int64
	err := d.db.QueryRowContext(ctx,
		"SELECT `tokens` FROM `user_ai_usage` WHERE `user_id` = ? AND `usage_date` = ?",
		userID, usageDate,
	).Scan(&tokens)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return tokens, nil
}
