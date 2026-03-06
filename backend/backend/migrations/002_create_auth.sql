-- Super Dev scaffold migration for module: auth
CREATE TABLE IF NOT EXISTS auth_items (
  id INTEGER PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
