-- user_ai_usage meters AI token spend per user per UTC day (usage_date is YYYY-MM-DD).
-- Used to enforce the free-tier daily token cap. No FK on user_id by convention.
CREATE TABLE user_ai_usage (
  user_id    INTEGER NOT NULL,
  usage_date TEXT    NOT NULL,
  tokens     BIGINT  NOT NULL DEFAULT 0,
  PRIMARY KEY (user_id, usage_date)
);
