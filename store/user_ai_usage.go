package store

import "context"

// IncrementUserAIUsage adds tokens to a user's AI usage for a UTC day (usageDate
// is YYYY-MM-DD), creating the row if it does not exist.
func (s *Store) IncrementUserAIUsage(ctx context.Context, userID int32, usageDate string, tokens int64) error {
	return s.driver.IncrementUserAIUsage(ctx, userID, usageDate, tokens)
}

// GetUserAIUsage returns the tokens a user has spent on the given UTC day, or 0
// if no usage is recorded.
func (s *Store) GetUserAIUsage(ctx context.Context, userID int32, usageDate string) (int64, error) {
	return s.driver.GetUserAIUsage(ctx, userID, usageDate)
}
