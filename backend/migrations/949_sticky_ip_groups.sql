ALTER TABLE proxy_ip_groups ADD COLUMN IF NOT EXISTS sticky_minutes INTEGER NOT NULL DEFAULT 0 CHECK (sticky_minutes = 0 OR sticky_minutes BETWEEN 20 AND 50);
