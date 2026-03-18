-- (idempotent version)

-- Grant replication
DO $$
BEGIN
    ALTER USER discore WITH REPLICATION;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Could not alter user (might need superuser)';
END $$;

-- Permissions (idempotent by default)
GRANT CONNECT ON DATABASE discore TO discore;
GRANT USAGE ON SCHEMA public TO discore;
GRANT SELECT ON public.members TO discore;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO discore;

-- Replica identity (idempotent)
ALTER TABLE public.members REPLICA IDENTITY FULL;


-- Create publication only if not exists
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_publication WHERE pubname = 'discore_publication'
    ) THEN
        CREATE PUBLICATION discore_publication FOR TABLE public.members;
    END IF;
END $$;
