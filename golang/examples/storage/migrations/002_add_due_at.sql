ALTER TABLE tasks ADD COLUMN due_at TIMESTAMPTZ;
CREATE INDEX tasks_due_at_idx ON tasks (due_at) WHERE NOT done;
