package adapters

const (
	// queryFetchNodes fetches tables, views, and materialized views with their size and row count
	// queryFetchNodes fetches tables, views, and materialized views with their size and row count
	queryFetchNodes = `
		SELECT
			ns.nspname AS schema_name,
			cl.relname AS table_name,
			CASE cl.relkind
				WHEN 'r' THEN 'TABLE'
				WHEN 'p' THEN 'TABLE (Partitioned)'
				WHEN 'v' THEN 'VIEW'
				WHEN 'm' THEN 'MATERIALIZED VIEW'
				ELSE 'OTHER'
			END AS type,
			pg_size_pretty(pg_total_relation_size(cl.oid)) AS size,
			CASE
				WHEN cl.relkind = 'r' THEN COALESCE(stat.n_live_tup, cl.reltuples, 0)
				WHEN cl.relkind = 'p' THEN COALESCE(cl.reltuples, 0)
				ELSE COALESCE(cl.reltuples, 0)
			END AS row_count
		FROM pg_class cl
		JOIN pg_namespace ns ON cl.relnamespace = ns.oid
		LEFT JOIN pg_stat_user_tables stat ON stat.relid = cl.oid
		WHERE cl.relkind IN ('r', 'p', 'v', 'm')
		  AND ns.nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast');
	`

	// queryFetchTriggers fetches trigger information
	queryFetchTriggers = `
		SELECT 
			n.nspname as schema,
			c.relname as table_name,
			t.tgname as trigger_name,
			p.proname as function_name,
			CASE t.tgtype & 1
				WHEN 1 THEN 'ROW'
				ELSE 'STATEMENT'
			END as level
		FROM pg_trigger t
		JOIN pg_class c ON t.tgrelid = c.oid
		JOIN pg_namespace n ON c.relnamespace = n.oid
		JOIN pg_proc p ON t.tgfoid = p.oid
		WHERE NOT t.tgisinternal
		AND n.nspname NOT IN ('information_schema', 'pg_catalog');
	`

	// queryDBVersion fetches the PostgreSQL version
	queryDBVersion = "SHOW server_version"

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
		WHERE v.relkind IN ('v', 'm')        -- Source is a View or Mat View
		  AND d.classid = 'pg_rewrite'::regclass
		  AND d.refclassid = 'pg_class'::regclass
		  AND d.deptype = 'n'
		  AND ref.relkind IN ('r', 'p', 'v', 'm') -- Target is a Table, Partitioned Table, View, or Mat View
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
	// queryFetchInheritance fetches table inheritance (partitions)
	queryFetchInheritance = `
		SELECT
			nm.nspname AS parent_schema,
			parent.relname AS parent_table,
			n.nspname AS child_schema,
			child.relname AS child_table
		FROM pg_inherits i
		JOIN pg_class parent ON i.inhparent = parent.oid
		JOIN pg_class child ON i.inhrelid = child.oid
		JOIN pg_namespace nm ON parent.relnamespace = nm.oid
		JOIN pg_namespace n ON child.relnamespace = n.oid
		WHERE nm.nspname NOT IN ('information_schema', 'pg_catalog')
		  AND n.nspname NOT IN ('information_schema', 'pg_catalog');
	`

	// queryFetchFunctionBody fetches the definition of a function
	queryFetchFunctionBody = `
		SELECT p.prosrc 
		FROM pg_proc p 
		JOIN pg_namespace n ON p.pronamespace = n.oid 
		WHERE p.proname = $1 
		  AND n.nspname = $2
	`

	// queryGetColumnAttNum fetches specific attribute number
	queryGetColumnAttNum = `
		SELECT a.attnum
		FROM pg_attribute a
		JOIN pg_class c ON a.attrelid = c.oid
		JOIN pg_namespace n ON c.relnamespace = n.oid
		WHERE n.nspname = $1 AND c.relname = $2 AND a.attname = $3
	`

	// queryColumnDependencies (Broader)
	queryColumnDependencies = `
		SELECT
			d.classid::regclass::text AS dependant_type,
			CASE 
				WHEN d.classid = 'pg_rewrite'::regclass THEN (
					SELECT ev_class::regclass::text FROM pg_rewrite WHERE oid = d.objid
				)
				WHEN d.classid = 'pg_trigger'::regclass THEN (
					SELECT tgname FROM pg_trigger WHERE oid = d.objid
				)
				WHEN d.classid = 'pg_constraint'::regclass THEN (
					SELECT conname FROM pg_constraint WHERE oid = d.objid
				)
				WHEN d.classid = 'pg_attrdef'::regclass THEN 'DEFAULT VALUE'
				WHEN d.classid = 'pg_class'::regclass THEN d.objid::regclass::text
				ELSE 'OID:' || d.objid::text
			END AS dependant_name,
			d.deptype::text,
			n.nspname AS schema_name
		FROM pg_depend d
		JOIN pg_attribute a ON d.refobjid = a.attrelid AND d.refobjsubid = a.attnum
		JOIN pg_class c ON d.refobjid = c.oid
		JOIN pg_namespace n ON c.relnamespace = n.oid
		WHERE c.relname = $1
		  AND n.nspname = $2
		  AND a.attname = $3
		  -- Removed strict deptype filtering to catch everything
	`

	// queryFKRefsByColumn finds FKs that reference this specific column
	queryFKRefsByColumn = `
		SELECT 
			con.conname,
			con.conrelid::regclass::text AS source_table
		FROM pg_constraint con
		JOIN pg_class c ON con.confrelid = c.oid
		JOIN pg_namespace n ON c.relnamespace = n.oid
		WHERE n.nspname = $1 
		  AND c.relname = $2
		  AND con.contype = 'f'
		  AND $3 = ANY(con.confkey)
	`

	// queryIndexesByColumn finds indexes using this column
	queryIndexesByColumn = `
		SELECT 
			ix.indexrelid::regclass::text AS index_name
		FROM pg_index ix
		JOIN pg_class c ON ix.indrelid = c.oid
		JOIN pg_namespace n ON c.relnamespace = n.oid
		WHERE n.nspname = $1
		  AND c.relname = $2
		  AND $3 = ANY(ix.indkey)
	`

	// queryScanFunctionsForColumn usage scans function bodies for column usage (regex/text search)
	queryScanFunctionsForColumn = `
		SELECT
			n.nspname AS schema_name,
			p.proname AS function_name,
			p.prosrc AS source_code
		FROM pg_proc p
		JOIN pg_namespace n ON p.pronamespace = n.oid
		WHERE n.nspname NOT IN ('information_schema', 'pg_catalog')
		  AND p.prosrc ILIKE '%' || $1 || '%'
	`

	// queryTableDependencies fetches objects that depend on a whole table
	queryTableDependencies = `
		SELECT
			d.classid::regclass::text AS dependant_type,
			CASE 
				WHEN d.classid = 'pg_rewrite'::regclass THEN (
					SELECT ev_class::regclass::text FROM pg_rewrite WHERE oid = d.objid
				)
				WHEN d.classid = 'pg_trigger'::regclass THEN (
					SELECT tgname FROM pg_trigger WHERE oid = d.objid
				)
				WHEN d.classid = 'pg_constraint'::regclass THEN (
					SELECT conname FROM pg_constraint WHERE oid = d.objid
				)
				WHEN d.classid = 'pg_type'::regclass THEN d.objid::regtype::text
				ELSE d.objid::regclass::text
			END AS dependant_name,
			d.deptype::text,
			n.nspname AS schema_name
		FROM pg_depend d
		JOIN pg_class c ON d.refobjid = c.oid
		JOIN pg_namespace n ON c.relnamespace = n.oid
		WHERE c.relname = $1
		  AND n.nspname = $2
		  -- Removed refobjsubid=0 to catch dependencies on columns too
	`

	// queryFKRefsByTable finds FKs pointing TO this table
	queryFKRefsByTable = `
		SELECT 
			con.conname,
			con.conrelid::regclass::text AS source_table
		FROM pg_constraint con
		JOIN pg_class fcl ON con.confrelid = fcl.oid
		JOIN pg_namespace fns ON fcl.relnamespace = fns.oid
		WHERE fns.nspname = $1 
		  AND fcl.relname = $2
		  AND con.contype = 'f' -- Foreign Key
	`
)
