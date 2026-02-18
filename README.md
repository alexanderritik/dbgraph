<div align="center">

![dbgraph logo](logo.svg)

</div>

[![Go Report Card](https://goreportcard.com/badge/github.com/alexanderritik/dbgraph)](https://goreportcard.com/report/github.com/alexanderritik/dbgraph)
[![Go Reference](https://pkg.go.dev/badge/github.com/alexanderritik/dbgraph.svg)](https://pkg.go.dev/github.com/alexanderritik/dbgraph)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/v/release/alexanderritik/dbgraph)](https://github.com/alexanderritik/dbgraph/releases)


**The missing schema dependency analyzer for databases.**

> **Note:** Currently supports **PostgreSQL**. MySQL and NoSQL support is coming soon!

`dbgraph` is a CLI tool designed for engineering teams who need to understand their database schema's web of dependencies instantly. It answers the question: *"If I drop this table or change this column, what breaks?"* without requiring a full code audit.

Unlike heavy GUI tools or passive log analyzers, `dbgraph` is an active, graph-based explorer that runs in your terminal, offering immediate insights into schema risks, query performance, and architectural hotspots.

![Demo](demo.gif)

---

## ⚡ Installation

Get started in seconds. No dependencies required.

### 🍎 macOS & Linux (via Curl)
```bash
curl -fsSL https://raw.githubusercontent.com/alexanderritik/dbgraph/main/install.sh | bash
```

### 🐹 Go Install
```bash
go install github.com/alexanderritik/dbgraph@latest
```

### 🐳 Docker
```bash
docker run --rm -e DBGRAPH_DB_URL="postgres://user:pass@host/db" alexanderritik/dbgraph summary
```

---

## 🎮 Command Reference

set `DBGRAPH_DB_URL` environment variable for seamless usage, or use `--db` flag.

| Feature | Command | Execution Example | Benefit |
| :--- | :--- | :--- | :--- |
| **Dependency Impact** | `impact` | `dbgraph impact users` | visualizes cascading effects (FKs, Views, Triggers) of changing a table. Prevents "oops" moments in production. |
| **Schema Simulation** | `simulate` | `dbgraph simulate --drop-column users.email` | **Dry-run** destructive changes. Tells you exactly which views or procedures will fail *before* you run the migration. |
| **Query Performance** | `top` | `dbgraph top --watch` | Real-time `htop` for your queries. Spot bottleneck queries instantly with live load metrics and execution frequency. |
| **Query Tracing** | `trace` | `dbgraph trace --query "SELECT * FROM users..."` | Runs `EXPLAIN (ANALYZE, BUFFERS)` and visualizes the execution path, cache hits, and I/O latency in a readable tree format. |
| **Architectural Summary** | `summary` | `dbgraph summary` | High-level ranking of your "God Objects" and riskiest tables based on centrality and connectedness. |
| **Full Analysis** | `analyze` | `dbgraph analyze` | Performs a deep health check: finds circular dependencies, missing indexes on FKs, and isolated schema islands. |

---

## 🔍 Deep Dive

### 1. Structural Impact Analysis
Avoid downtime caused by unintended cascades. `dbgraph` builds a Directed Acyclic Graph (DAG) of your schema constraints.

```bash
$ dbgraph impact orders
🔍 DB: production | Target: public.orders (1.2m rows)
--------------------------------------------------------------------------------
📊 IMPACT RADIUS: 3 Levels Deep
└── 📥 public.orders
    ├── 📥 public.order_items [FK: fk_order] (CASCADE) ⚠️
    │   └── 👁️  public.finance_report_view
    └── ⚡ trigger_update_inventory
```

### 2. "What-If" Simulations
Planning a refactor? Simulate it first.
```bash
$ dbgraph simulate --drop-column users.country_code
🧪 Simulating DROP COLUMN on public.users.country_code...

public.users.country_code
└── 👁️  public.v_user_demographics (View Dependency)
└── 📜 public.get_user_region (Function Body Usage)
```

### 3. Real-Time Monitoring
Debug performance issues live during incidents without leaving your terminal.
```bash
$ dbgraph top --watch --sort total
RANK  LOAD %  TIME (ms)  CALLS   QUERY
1     45.2%   1204.50    502     SELECT * FROM orders WHERE...
2     12.0%   320.10     10      UPDATE inventory SET...
```

---

## 🆚 Comparison

| Feature | **dbgraph** | **GUI Clients (DataGrip/DBeaver)** | **Monitoring (Datadog/pgBadger)** |
| :--- | :--- | :--- | :--- |
| **Focus** | **Schema Dependencies & Risk** | SQL Execution & Browsing | Historical Performance Trends |
| **Interface** | CLI (Scriptable, Fast) | Heavy UI | Web Dashboards |
| **Dependency Analysis** | **Graph-based (Recursive)** | Basic ER Diagrams | None |
| **Simulation** | ✅ **Dry-run logic** | ❌ Run it and pray | ❌ Post-mortem only |
| **Overhead** | **Zero/Negligible** (Metadata queries) | Low | Varies (Agent based) |

## 🛡️ Production Safety
*   **Read-Only by Design**: `dbgraph` never modifies your data. It runs `SET TRANSACTION READ ONLY` for safety.
*   **Metadata First**: We query `pg_catalog` system views for 99% of operations, keeping load on your tables near zero.

---

## 🔮 Roadmap & Future

We are just getting started! `dbgraph` is built to evolve, and we want to support your entire stack.

**Coming Soon:**
*   **Multi-Database Support**: MySQL, SQLite, and NoSQL are on the roadmap.
*   **New Features**: We are open to community requests!

Missing something critical? [Open an issue](https://github.com/alexanderritik/dbgraph/issues) or contribute a PR. We love building with the community.

## License

Distributed under the MIT License. See [LICENSE](LICENSE) for more information.
