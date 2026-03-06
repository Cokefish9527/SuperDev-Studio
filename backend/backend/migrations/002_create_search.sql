-- Super Dev scaffold migration for module: search
CREATE TABLE IF NOT EXISTS search_items (
  id INTEGER PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
