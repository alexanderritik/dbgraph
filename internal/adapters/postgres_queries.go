package adapters

const (
	// queryFetchNodes fetches tables, views, and materialized views with their size and row count
	queryFetchNodes = `
		SELECT
			ns.nspname AS schema_name,
			cl.relname AS table_name,
			CASE cl.relkind
				WHEN 'r' THEN 'TABLE'
				WHEN 'v' THEN 'VIEW'
				WHEN 'm' THEN 'MATERIALIZED VIEW'
				ELSE 'OTHER'
			END AS type,
			pg_size_pretty(pg_total_relation_size(cl.oid)) AS size,
			COALESCE(cl.reltuples, 0) AS row_count
		FROM pg_class cl
		JOIN pg_namespace ns ON cl.relnamespace = ns.oid
		WHERE cl.relkind IN ('r', 'v', 'm')
		  AND ns.nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast');
	`

	// queryFetchIndexes fetches index columns for tables to detect missing indexes
	queryFetchIndexes = `
		select
			ns.nspname as schema_name,
			t.relname as table_name,
			(
				select array_agg(a.attname order by array_position(ix.indkey, a.attnum))
				from pg_attribute a
				where a.attrelid = t.oid and a.attnum = any(ix.indkey)
			) as columns
		from pg_index ix
		join pg_class t on ix.indrelid = t.oid
		join pg_namespace ns on t.relnamespace = ns.oid
		where ns.nspname not in ('information_schema', 'pg_catalog', 'pg_toast');
	`

	// queryFetchForeignKeys fetches foreign key constraints and their metadata
	queryFetchForeignKeys = `
		SELECT
			ns.nspname AS table_schema,
			cl.relname AS table_name,
			fns.nspname AS foreign_table_schema,
			fcl.relname AS foreign_table_name,
			con.conname AS constraint_name,
			CASE con.confdeltype
				WHEN 'a' THEN 'NO ACTION'
				WHEN 'r' THEN 'RESTRICT'
				WHEN 'c' THEN 'CASCADE'
				WHEN 'n' THEN 'SET NULL'
				WHEN 'd' THEN 'SET DEFAULT'
			END AS delete_rule,
			(
				SELECT array_agg(a.attname ORDER BY array_position(con.conkey, a.attnum))
				FROM pg_attribute a
				WHERE a.attrelid = cl.oid AND a.attnum = ANY(con.conkey)
			) AS fk_columns
		FROM pg_constraint con
		JOIN pg_class cl ON con.conrelid = cl.oid
		JOIN pg_namespace ns ON cl.relnamespace = ns.oid
		JOIN pg_class fcl ON con.confrelid = fcl.oid
		JOIN pg_namespace fns ON fcl.relnamespace = fns.oid
		WHERE con.contype = 'f'
		  AND ns.nspname NOT IN ('information_schema', 'pg_catalog')
		  AND fns.nspname NOT IN ('information_schema', 'pg_catalog');
	`

	// queryFetchViews fetches view dependencies
	queryFetchViews = `
		SELECT
			v.relnamespace::regnamespace::text AS view_schema,
			v.relname AS view_name,
			ref.relnamespace::regnamespace::text AS target_schema,
			ref.relname AS target_name
		FROM pg_depend d
		JOIN pg_rewrite r ON d.objid = r.oid
		JOIN pg_class v ON r.ev_class = v.oid
		JOIN pg_class ref ON d.refobjid = ref.oid
		WHERE v.relkind = 'v'        -- Source is a View
		  AND d.classid = 'pg_rewrite'::regclass
		  AND d.refclassid = 'pg_class'::regclass
		  AND d.deptype = 'n'
		  AND ref.relkind IN ('r', 'v') -- Target is a Table or View
		  AND v.oid != ref.oid       -- Exclude self-references logic if any
		  AND v.relnamespace::regnamespace::text NOT IN ('information_schema', 'pg_catalog')
		  AND ref.relnamespace::regnamespace::text NOT IN ('information_schema', 'pg_catalog');
	`

	// queryActiveLocks counts active locks in the database
	queryActiveLocks = "SELECT count(*) FROM pg_locks WHERE granted = true"

	// queryMaxConns fetches the max_connections setting
	queryMaxConns = "SELECT setting::int FROM pg_settings WHERE name = 'max_connections'"

	// queryUsedConns counts active connections
	queryUsedConns = "SELECT count(*) FROM pg_stat_activity"

	// queryLongestRunning fetches the longest running active query
	queryLongestRunning = `
		SELECT 
			EXTRACT(EPOCH FROM (now() - query_start))::float, 
			pid 
		FROM pg_stat_activity 
		WHERE state = 'active' 
		ORDER BY query_start ASC 
		LIMIT 1
	`
)
