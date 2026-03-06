-- Super Dev scaffold migration for module: analytics
CREATE TABLE IF NOT EXISTS analytics_items (
  id INTEGER PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
