---
name: go-release
description: >-
  用於在此 Go 儲存庫中準備 develop 到 main 的 release、產生 release notes、建議 Semantic Version tag，
  並處理 main 上的 hotfix 發布流程。
---

# go-release

## 快速導覽

- [概覽](#概覽)
- [使用時機](#使用時機)
- [前置檢查](#前置檢查)
- [工作流程](#工作流程)
- [Release Notes 規則](#release-notes-規則)
- [Tag 與 Push 規則](#tag-與-push-規則)
- [收尾問題](#收尾問題)
- [常見錯誤](#常見錯誤)

## 概覽

將 `develop` 的穩定變更發布到 `main`。先做 branch / working tree / remote / `ci-test` 檢查，再用 `python scripts/release-notes.py` 收集上一個 tag 以來的 Conventional Commits，交給乾淨 context 的 subagent 濃縮成 release notes，最後才建議版本號、merge 到 `main` 並建立 lightweight tag。

預設流程是 `develop -> main`。只有當使用者明確在 `main` 上處理 hotfix 時，才改走 hotfix 路徑。

[返回開頭](#快速導覽)

## 使用時機

- 使用者要求發布新版本
- 使用者要求將 `develop` merge 到 `main` 並打 tag
- 使用者要求整理上一個 tag 之後的 release notes
- 使用者在 `main` 上處理 hotfix，並需要建立新 tag

[返回開頭](#快速導覽)

## 前置檢查

1. 檢查 `git branch --show-current`。
2. 正常 release 必須從 `develop` 開始；hotfix 只允許從 `main` 開始。
3. 檢查 `git status --short`；若工作樹不乾淨，停止並要求先 commit 或 stash。
4. 執行 `git fetch --all --tags --prune`。
5. 若正常 release 且 `origin/develop` 領先本地，停止並要求先同步 `develop`。
6. 執行 ci-test；若失敗，停止並指向 `test-output/ci-test/`。
   - Unix / macOS：`bash scripts/ci-test.sh`
   - Windows：`pwsh -File .\scripts\ci-test.ps1`
7. 若自上一個 tag 以來沒有新 commits，停止並明說目前沒有可發布內容。

[返回開頭](#快速導覽)

## 工作流程

1. 完成所有前置檢查。
2. 執行 `python scripts/release-notes.py`，讓它將原始 commit 資料寫入 `.tmp/release/raw-commits.json`。
3. 啟動一個乾淨 context 的 subagent，只讀 `.tmp/release/raw-commits.json`，將結果濃縮成 `.tmp/release/release-notes.md`。
4. 先向使用者展示濃縮後的 release notes，讓使用者確認內容。
5. 只有在 release notes 確認後才建議版本號：
   - 有 breaking changes：建議 MAJOR bump
   - 無 breaking changes 但有 features：建議 MINOR bump
   - 其餘情況：建議 PATCH bump
6. 使用者可接受或覆寫版本號；若格式不符 `vMAJOR.MINOR.PATCH`，要求重新輸入。
7. 正常 release 路徑（Merge 階段）：
   - `git checkout main`
   - `git pull origin main`
   - 執行 `git merge --no-ff develop -m "release: vX.Y.Z"`
   - 若產生 conflict：立即通知使用者、列出衝突檔案、協助解決或明確建議 `git merge --abort`；**conflict 未解決或 merge 未完成前，不得繼續下一步**
8. 正常 release 路徑（Tag 階段，僅在 merge 成功完成後執行）：
   - `git tag vX.Y.Z`
9. Hotfix 路徑（當前分支是 `main`）：
   - 不做 merge
   - 直接 `git tag vX.Y.Z`
10. 完成後以一個明確的問題結束，只提供三個選項：
    - 推送 `main` + tags
    - 回到 `develop` 並 merge `main`
    - 以上都做

[返回開頭](#快速導覽)

## Release Notes 規則

- Python 腳本只做 deterministic 的 commit 收集與分組，不負責濃縮語意。
- Subagent 讀取 `.tmp/release/raw-commits.json` 後，應將 notes 按 `Breaking Changes`、`Features`、`Bug Fixes`、`Performance`、`Other Improvements` 排序。
- `merge:`、`release:` 與 Git 自動產生的 merge commits 不應進入 release notes。
- `unparseable` commits 必須保留，供使用者人工檢查。

[返回開頭](#快速導覽)

## Tag 與 Push 規則

- Tag 固定使用 lightweight tag：`git tag vX.Y.Z`
- Tag 必須打在正常 release 的 merge commit 上，或 hotfix 路徑的 `main` HEAD 上
- 不要自動建立 annotated tag
- 不要自動推送；只有在使用者選擇後才執行 `git push origin main --tags`

[返回開頭](#快速導覽)

## 收尾問題

- 正常 release：

  ```
  Release vX.Y.Z 完成。下一步請選擇：
  1. 推送 `main` + tags
  2. 回到 `develop` 並 merge `main`（同步 merge commit）
  3. 以上都做
  ```

- Hotfix release：

  ```
  Hotfix vX.Y.Z 完成。下一步請選擇：
  1. 推送 `main` + tags
  2. 將 `main` merge 回 `develop`
  3. 以上都做
  ```

[返回開頭](#快速導覽)

## 常見錯誤

- 在 feature branch 上直接做 release
- release notes 還沒確認就先決定版本號
- 把 `merge:` / `release:` 這類流程 commit 當成使用者可見變更
- 在未執行 `ci-test` 的情況下直接 merge 到 `main`
- 未經使用者同意就自動推送 `main` 或 tags

[返回開頭](#快速導覽)
