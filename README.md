# dbgraph

`dbgraph` is a high-fidelity dependency analysis tool for PostgreSQL. It helps you visualize table dependencies, analyze schema impact, and detect structural risks like missing indexes and dangerous cascade rules.

## Key Features

*   **Impact Analysis**: Visualize exactly what breaks when you modify a table or view.
*   **Safety First**: Uses read-only transactions. **Zero impact on database performance**.
*   **Dependency Visualization**: Tree-based view of Foreign Keys, Views, and Trigger dependencies.
*   **Risk Detection**: Automatically flags `ON DELETE CASCADE` risks and missing index warnings.
*   **Real-Time Metrics**: Displays active locks, connection saturation, and row estimates during analysis.

## Safety & Performance

`dbgraph` is designed to be safe for production environments.

*   **Read-Only**: All queries run in read-only mode. The tool never modifies your schema or data.
*   **Metadata Only**: The tool queries `pg_catalog` (system tables) to build the graph. It does **not** scan your actual table data, ensuring minimal load.
*   **Lightweight**: Row counts are estimates (`reltuples`) from system statistics, preventing expensive `COUNT(*)` operations on large tables.

## Installation

### Quick Install (Mac & Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/alexanderritik/dbgraph/main/install.sh | bash
```

### Manual Install

Download the binary for your OS from the [Releases](https://github.com/alexanderritik/dbgraph/releases) page.

```bash
# Example for macOS ARM64
curl -L -o dbgraph https://github.com/alexanderritik/dbgraph/releases/latest/download/dbgraph-darwin-arm64
chmod +x dbgraph
sudo mv dbgraph /usr/local/bin/
```

### From Source

```bash
go install github.com/alexanderritik/dbgraph@latest
```

## Usage

### Impact Analysis

Analyze the downstream impact of changing a specific table:

```bash
dbgraph impact users --db="postgres://user:pass@localhost:5432/dbname"
```

**Output Example:**

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
[Med] Missing Index: 'public.order_items(order_id)' is not indexed. Cascade/Delete operations will be slow.
```

### Schema Visualization

Export your entire schema dependency graph to DOT format for visualization:

```bash
dbgraph analyze --db="..." > schema.dot
```

## License

MIT
