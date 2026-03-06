-- Super Dev scaffold migration for module: operation
CREATE TABLE IF NOT EXISTS operation_items (
  id INTEGER PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
