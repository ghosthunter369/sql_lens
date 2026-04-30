# SQL Lens

SQL 可视化解析工具。将复杂 SQL 一键解析为表关系图、字段来源、JOIN 关系、WHERE 条件树和风险提示，帮助开发者 10 秒内看懂任意复杂 SQL。

## 功能

- **SQL 解析** — 支持 SELECT / JOIN / WHERE / GROUP BY / ORDER BY / LIMIT，完整兼容 MySQL 反引号标识符
- **日志提取** — 直接粘贴 MyBatis、Laravel、ThinkPHP 日志，自动提取 SQL 并回填参数
- **表关系图** — 可拖拽缩放的交互式表节点图，直观展示表之间的 JOIN 关系和关联字段
- **字段来源** — 逐字段列出输出名、来源表、来源字段、别名和表达式类型（普通字段/函数/聚合/CASE）
- **WHERE 条件树** — 嵌套 AND/OR 树形展示，一眼看清条件层级和逻辑关系
- **JOIN 详情** — 列出每个 JOIN 的类型、左右表及 ON 条件
- **风险检测** — 自动识别全表扫描、隐式类型转换、SELECT *、缺少 WHERE 条件等潜在问题

## 快速启动

确保已安装 Docker 和 Docker Compose。

```bash
cd sqlGen
docker compose up -d
```

访问 **http://localhost:3008**，在左侧编辑器中粘贴 SQL，点击分析即可。

## 本地开发

```bash
# 后端（需要 Go 1.22+）
cd backend
go run ./cmd/server

# 前端（需要 Node.js 18+）
cd frontend
npm install
npm run dev
```

前端 dev server 运行在 `:5173`，API 请求自动代理到本地后端 `:8080`。
