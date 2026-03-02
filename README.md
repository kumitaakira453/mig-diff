# mig-diff

Djangoマイグレーションのブランチ間差分を表示し、ロールバックコマンドを生成するCLIツール。

[mig-tree](https://github.com/kumitaakira453/mig-tree)のCLI機能を独立したツールとして提供。

## インストール

```bash
go install github.com/kumitaakira453/mig-diff@latest
```

## 使い方

```bash
# targetブランチとの差分を表示
mig-diff <target-branch>

# 例: mainブランチとの差分
mig-diff main

# ヘルプ
mig-diff --help
```

## 出力例

```
Comparing migrations: feature/new-feature → main
──────────────────────────────────────────────────

App: organization
  Current branch has migrations not in target:
    x 0005_add_new_field
    x 0004_update_model
  Rollback to: 0003_existing_migration

App: bff_main
  No rollback needed (branches have same migrations)

Commands to run:
╭───────────────────────────────────────────────────────────────╮
│ python manage.py migrate organization 0003_existing_migration │
╰───────────────────────────────────────────────────────────────╯

Commands copied to clipboard!

Run these commands? [y/N]:
```

## 設定ファイル

- `.mig-diff.yaml`（リポジトリ固有）
- `~/.config/mig-diff/config.yaml`（グローバル）

### 設定例

```yaml
apps:
  - organization
  - bff_main
  - shared

migrate_cmd: "python manage.py migrate"
```

## ライセンス

MIT
