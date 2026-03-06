-- Super Dev scaffold migration for module: content
CREATE TABLE IF NOT EXISTS content_items (
  id INTEGER PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
