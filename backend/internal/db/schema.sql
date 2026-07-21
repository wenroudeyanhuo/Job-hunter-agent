CREATE TABLE IF NOT EXISTS jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    company TEXT NOT NULL,
    title TEXT NOT NULL,
    city TEXT NOT NULL DEFAULT '',
    direction_tags TEXT NOT NULL DEFAULT '[]',
    description TEXT NOT NULL DEFAULT '',
    source_name TEXT NOT NULL DEFAULT '',
    source_url TEXT NOT NULL DEFAULT '',
    apply_url TEXT NOT NULL DEFAULT '',
    published_at TIMESTAMP NULL,
    deadline_at TIMESTAMP NULL,
    discovered_at TIMESTAMP NOT NULL,
    match_score INTEGER NOT NULL DEFAULT 0,
    recommend_reasons TEXT NOT NULL DEFAULT '[]',
    penalty_reasons TEXT NOT NULL DEFAULT '[]',
    status TEXT NOT NULL DEFAULT 'new',
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_company_title_city ON jobs(company, title, city);
CREATE INDEX IF NOT EXISTS idx_jobs_discovered_at ON jobs(discovered_at);

CREATE TABLE IF NOT EXISTS companies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    aliases TEXT NOT NULL DEFAULT '[]',
    category TEXT NOT NULL DEFAULT '',
    enabled INTEGER NOT NULL DEFAULT 1,
    priority INTEGER NOT NULL DEFAULT 0,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS job_sources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,
    url TEXT NOT NULL DEFAULT '',
    company_id INTEGER NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    category TEXT NOT NULL DEFAULT 'general',
    parser_type TEXT NOT NULL DEFAULT 'generic',
    last_run_at TIMESTAMP NULL,
    health_status TEXT NOT NULL DEFAULT 'unknown',
    health_reason TEXT NOT NULL DEFAULT '',
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    last_success_at TIMESTAMP NULL,
    last_failure_at TIMESTAMP NULL,
    last_found_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(company_id) REFERENCES companies(id)
);

CREATE TABLE IF NOT EXISTS job_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    trigger_type TEXT NOT NULL,
    started_at TIMESTAMP NOT NULL,
    finished_at TIMESTAMP NULL,
    status TEXT NOT NULL,
    sources_total INTEGER NOT NULL DEFAULT 0,
    sources_success INTEGER NOT NULL DEFAULT 0,
    sources_failed INTEGER NOT NULL DEFAULT 0,
    jobs_found INTEGER NOT NULL DEFAULT 0,
    jobs_created INTEGER NOT NULL DEFAULT 0,
    jobs_duplicated INTEGER NOT NULL DEFAULT 0,
    manual_check_count INTEGER NOT NULL DEFAULT 0,
    error_summary TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS job_run_sources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_run_id INTEGER NOT NULL,
    source_name TEXT NOT NULL DEFAULT '',
    source_url TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    jobs_found INTEGER NOT NULL DEFAULT 0,
    jobs_created INTEGER NOT NULL DEFAULT 0,
    jobs_duplicated INTEGER NOT NULL DEFAULT 0,
    jobs_filtered INTEGER NOT NULL DEFAULT 0,
    manual_check_count INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(job_run_id) REFERENCES job_runs(id)
);

CREATE INDEX IF NOT EXISTS idx_job_run_sources_run_id ON job_run_sources(job_run_id);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS agent_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    level TEXT NOT NULL DEFAULT 'info',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_agent_events_created_at ON agent_events(created_at);

CREATE TABLE IF NOT EXISTS agent_tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_date TEXT NOT NULL,
    kind TEXT NOT NULL,
    title TEXT NOT NULL,
    detail TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'open',
    priority INTEGER NOT NULL DEFAULT 0,
    count INTEGER NOT NULL DEFAULT 1,
    subject_id INTEGER NOT NULL DEFAULT 0,
    job_id INTEGER NOT NULL DEFAULT 0,
    source_id INTEGER NOT NULL DEFAULT 0,
    action TEXT NOT NULL DEFAULT '',
    completion_reason TEXT NOT NULL DEFAULT '',
    snoozed_until TIMESTAMP NULL,
    escalated_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    UNIQUE(task_date, kind, subject_id)
);

CREATE INDEX IF NOT EXISTS idx_agent_tasks_date_status ON agent_tasks(task_date, status, priority);
