-- Super Dev scaffold migration for module: experience
CREATE TABLE IF NOT EXISTS experience_items (
  id INTEGER PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
