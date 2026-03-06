-- Super Dev scaffold migration for module: core
CREATE TABLE IF NOT EXISTS core_items (
  id INTEGER PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
