-- Super Dev scaffold migration for module: profile
CREATE TABLE IF NOT EXISTS profile_items (
  id INTEGER PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
