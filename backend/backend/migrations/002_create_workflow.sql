-- Super Dev scaffold migration for module: workflow
CREATE TABLE IF NOT EXISTS workflow_items (
  id INTEGER PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
