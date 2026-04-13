ALTER TABLE tasks
    ADD COLUMN periodicity JSONB,
    ADD COLUMN periodic_source_id BIGINT REFERENCES tasks(id) ON DELETE SET NULL;

CREATE INDEX idx_tasks_periodicity ON tasks USING GIN (periodicity) WHERE periodicity IS NOT NULL;
CREATE INDEX idx_tasks_periodic_source ON tasks (periodic_source_id) WHERE periodic_source_id IS NOT NULL;
