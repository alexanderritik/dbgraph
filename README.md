# 🔥 dbgraph — The Swiss Army Knife for PostgreSQL Schemas

`dbgraph` is not just another CLI — it’s a nuclear-powered dependency analyzer for Postgres.

Ever deleted a table and accidentally nuked 10 other tables? Or spent hours figuring out why that one query is slow? `dbgraph` shows you the entire domino effect in one glance — safely, instantly, and in a way even your DBA fears to dream about.

## ⚡ What Makes dbgraph Insanely Useful

| Feature | Why You’ll Love It |
| :--- | :--- |
| **Impact Analysis** | See exactly what breaks if you change a table, column, or view — cascade deletes, dependent views, triggers. |
| **Risk Detection** | ⚠️ Flags dangerous `ON DELETE CASCADE` rules and missing indexes before disaster strikes. |
| **Simulation Mode** | Virtually delete or modify tables without touching your data. Dry-run the apocalypse. |
| **Real-Time Metrics** | Active locks, connection saturation, and row estimates — all while you explore your schema. |
| **Hotspot & Query Tracing** | Map slow queries to tables, see index usage, and find your DB’s choke points. |
| **Schema Diffs** | Compare branches or snapshots — see what changed, what’s risky, and what will break. |

## 🎯 Safety & Performance — Production-Proof (Zero Impact Promise)

*   **Zero Performance Impact** — It’s lighter than a neutrino. Your database won’t even know we’re there. It has about as much impact on your DB performance as a butterfly landing on a tank. 🦋
*   **Read-Only Transactions** — We use `SET TRANSACTION READ ONLY`. `dbgraph` couldn't modify your data even if it wanted to. It’s strictly look-but-don’t-touch.
*   **Metadata-Only Queries** — We query `pg_catalog` (system views), not your 500GB tables. We don't run `COUNT(*)`. We just politely ask Postgres, "Hey, how big is this table?" and Postgres whispers back the stats.

> “I ran `dbgraph impact orders` on our production DB — saw the cascade to 15 tables before I touched a single row. Saved us from a potential outage.” – Anonymous Senior Engineer

## 🚀 Quick Start — Install in 10s

### Mac & Linux (curl magic)

```bash
curl -fsSL https://raw.githubusercontent.com/alexanderritik/dbgraph/main/install.sh | bash
```

### From Source

```bash
go install github.com/alexanderritik/dbgraph@latest
```

### Docker-Friendly

```bash
docker run --rm -e DATABASE_URL=postgres://user:pass@localhost:5432/dbname alexanderritik/dbgraph impact users
```

## 🌳 Usage Example — Impact Analysis

```bash
dbgraph impact users --db="postgres://user:pass@localhost:5432/dbname"
```

**Output Preview:**

```text
🔍 DB: production | Target: public.users (1.2m rows) | Active Locks: 4
--------------------------------------------------------------------------------

📊 IMPACT RADIUS: 3 Levels Deep [Load: System Normal]
Total Affected Objects: 12 (8 TABLEs, 4 VIEWs)

TREE VIEW
public.users (1.2m rows)
└── 📥 public.orders (4.5m rows) [FK: fk_user] (CASCADE)
    ├── 📥 public.order_items (15m rows) [FK: fk_order] (CASCADE)
    │   └── 👁️  public.high_value_orders (View)
    └── 👁️  public.user_order_summary (View)

⚠️  STRUCTURAL WARNINGS
[High] Cascade Delete: Deleting 'public.orders' will recursively delete objects in 'public.order_items'.
[Med] Missing Index: 'public.order_items(order_id)' is not indexed.
```

## 🌐 Schema Visualization

Export your schema dependency graph to DOT or Graphviz:

```bash
dbgraph analyze --db="..." > schema.dot
dot -Tpng schema.dot -o schema.png
```

## 🎁 For Senior Engineers

*   **God Table Detection**: Tables with 20+ inbound FKs get flagged
*   **Hotspots**: Shows sequential scan-heavy tables
*   **Simulate & Diff**: Try changes safely and compare schema snapshots
*   **Infer Relationships**: Detect “hidden” FKs like `user_id` → `users.id`

## 🛡 License

MIT – Do whatever you want, just don’t blame me if your cascade nukes your production 😎
