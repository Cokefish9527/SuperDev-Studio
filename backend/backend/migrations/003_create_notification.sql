-- Super Dev scaffold migration for module: notification
CREATE TABLE IF NOT EXISTS notification_items (
  id INTEGER PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
