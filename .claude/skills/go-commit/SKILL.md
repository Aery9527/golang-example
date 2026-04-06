---
name: go-commit
description: 用於在此 Go 儲存庫中準備 git commit，尤其是當使用者要求提交變更、選擇提交邊界、暫存檔案、撰寫 Conventional Commit 訊息，或完成連結工作樹提交流程時使用。
---

# go-commit

## 概覽

將目前工作樹轉化為一個高品質 commit。優先考量可用於 release note 的歷史紀錄：subject 應描述功能、行為或架構意義，而非檔案編輯操作。

本地提交保持快速。除非使用者明確要求，否則不要為了建立本地 commit 而執行 `ci-test` 或 `dev-test`。若此技能接著要實際執行 `git push`，先確認 repo-local hooks 已安裝，讓儲存庫的 `pre-push` hook 真正接手推送前驗證。

若目前 checkout 是連結工作樹，不要在本地 commit 就停止。完成後詢問該工作樹分支接下來應如何整合。

## 使用時機

- 使用者要求提交目前的變更
- 使用者需要 commit 訊息或協助暫存正確的檔案
- 使用者想將混合的變更拆分為獨立的 commit
- 使用者希望 commit 歷史可整理成 release note
- 使用者在連結 git 工作樹中完成工作，並需要本地 commit 後的下一個 Git 步驟

此技能不適用於獨立的 hook 偵錯；但若它自己要實際執行 `git push`，必須先確認 repo-local hooks 已安裝，必要時先執行儲存庫提供的安裝腳本。

## 工作流程

1. 檢查 `git status --short`、已暫存差異和未暫存差異。
2. 判斷變更是否代表單一高層次意圖。若否，停下來並提議分開 commit。
3. 僅暫存屬於同一意圖的檔案。
4. 若某一檔案包含多個意圖的 hunk，使用 `git add -p` 確保每個 commit 保持連貫性。
5. 選擇 `type` 和可選的 `scope`。
6. 以英文、祈使語氣、高層次措辭撰寫 `type(scope): summary`。
7. 若 diff 只揭示底層實作細節而無法可靠推斷高層次意圖，在最終確定訊息前先問一個明確的問題。
8. 僅在有遷移說明、重大變更或 issue 參考時才加入 body 或 footer。
9. 建立本地 commit。
10. 在建議下一個 Git 步驟前，偵測目前 checkout 是否為連結工作樹。使用 `.git` 間接指向或 `git worktree list --porcelain`；不要單憑目錄命名猜測。
11. 只有當此技能準備親自執行 `git push` 時，才檢查 repo-local hooks；若只是建議使用者下一步可推送，保留提醒即可，不要主動變更 hook 設定。
12. 實際執行 `git push` 前，先檢查 `git config --get core.hooksPath` 是否為 `.githooks`。
13. 若未安裝 repo-local hooks，在 Windows 使用 `pwsh -File .\scripts\install-git-hooks.ps1`；在 Bash 環境使用 `bash scripts/install-git-hooks.sh`。
14. 安裝後再次確認 `core.hooksPath` 指向 `.githooks`；若安裝失敗或確認失敗，停止 push 並明確回報，不要假裝 `pre-push` 仍會生效。
15. 只有在 hooks 已確認可用後才執行 `git push`，並提醒 `pre-push` 將執行 `ci-test`。
16. Git 可確認目前 checkout 是否為連結工作樹及其 checkout 的分支，但不能可靠地保留工作樹最初建立時的來源分支。只有當使用者或 session 情境已明確提供時，才將來源分支視為已知。
17. 若目前 checkout 是連結工作樹，以一個明確的問題結束，提供恰好三個後續步驟選項：rebase 到已知來源分支、rebase 到其他分支、或將此分支推送到其遠端。
18. 若使用者選擇來源分支 rebase 路徑而來源分支尚未知曉，在建議指令前先問一個明確的後續問題確認分支名稱。
19. 若目前 checkout 不是連結工作樹，除非使用者明確要求提交後整合協助，否則不要加入 rebase 或推送選單。
20. 若使用者已明確告知來源分支，在後續問題中使用該確切分支名稱。
21. 絕對不要從主要 checkout 分支、其他兄弟工作樹、或 `git worktree list` 中第一個出現的分支推斷來源分支；那些資料顯示的是目前 checkout，而非工作樹的建立來源。
22. 不要將三選項工作樹問題替換為立即詢問來源分支或基礎 commit 的問題。只有在使用者選擇來源分支 rebase 路徑後才詢問來源分支。
23. 除非 `.git` 指向 `.git/worktrees/...` 或 `git worktree list --porcelain` 將目前路徑顯示為非主要工作樹條目，否則將儲存庫根目錄的 checkout 視為主要 checkout。

## 快速參考

| 使用 `type` | 時機 |
| --- | --- |
| `feat` | 新功能或對外可見的行為 |
| `fix` | 修復錯誤或修正行為 |
| `refactor` | 不改變行為的內部重構 |
| `docs` | 僅變更文件 |
| `test` | 測試覆蓋率或測試工作流程變更 |
| `chore` | 非功能或修復的維護工作 |
| `perf` | 效能改善 |
| `build` | 相依套件、打包或建構系統變更 |
| `ci` | hook、自動化或 CI 工作流程變更 |

## Push 前 hook 檢查

- 只有在此技能要親自執行 `git push` 時才做 hook 檢查；單純提供「下一步可以 push」的建議時不要執行安裝腳本。
- 以 `git config --get core.hooksPath` 驗證 repo-local hooks 是否已安裝；預期值是 `.githooks`。
- 若缺少 hook，先執行 `scripts/install-git-hooks.ps1` 或 `scripts/install-git-hooks.sh`，再重新檢查設定。
- 若安裝或重新檢查失敗，停止 push 並直接說明 `pre-push` 無法保證會被觸發。

## 工作樹後續問題

當目前 checkout 是連結工作樹時，在本地 commit 後使用此結束問題：

- 來源分支已知：`此工作樹分支的下一步：rebase 到 <source-branch>、rebase 到其他分支，或將此分支推送到遠端？`
- 來源分支未知：`此工作樹分支的下一步：rebase 到其來源分支、rebase 到其他分支，或將此分支推送到遠端？`

只有在來源分支已從使用者或 session 情境中確認時，才將第一個選項替換為具體分支名稱。

若使用者已命名來源分支，優先使用該確切分支名稱，而非從儲存庫佈局猜測。

請依照此順序：

| 情境 | 下一個問題 |
| --- | --- |
| 主要 checkout 或非工作樹分支 | 除非使用者要求提交後整合協助，否則無工作樹後續選單 |
| 連結工作樹，來源分支已知 | 詢問三選項選單並命名該確切來源分支 |
| 連結工作樹，來源分支未知 | 詢問同樣的三選項選單，但保持來源分支選項為通用描述 |
| 使用者選擇來源分支 rebase 且來源分支仍未知 | 詢問一個明確的後續問題確認來源分支名稱 |

## 訊息規則

- 當受影響的範圍明確時保留 `scope`。
- 為未來的 release note 優化 subject。
- 優先描述對使用者或操作者有何變化，而非哪些檔案移動了。
- 避免使用 `update`、`tweak`、`misc`、`fix stuff` 等低品質摘要。
- 避免以檔案為主體的 subject，如 `update writer.go and rotate.go`。
- 若意圖不明確，優先進行一次明確的澄清，而非編造誤導性的訊息。
- 在連結工作樹中，將提交後的分支決策視為一個獨立的明確問題，而非默默假設 rebase 或推送。
- 若使用者已命名來源分支，優先使用其陳述的分支，而非任何儲存庫啟發式猜測。
- 保持三選項工作樹選單在前；只有在使用者選擇該路徑後才詢問來源分支。
- 實際執行 `git push` 前，先處理 repo-local hook 安裝；若只是描述後續建議，則不要提前修改 hook 設定。

## 常見錯誤

- 因時間壓力將不相關的目標混入一個 commit
- 撰寫以檔案命名而非行為描述的 subject
- 使用低品質訊息如 `chore: update files`
- 在使用者未要求時於本地 commit 前執行測試
- 在 fresh clone 或未安裝 hooks 的 checkout 中直接 push，卻沒有先確認 `core.hooksPath` 是否指向 `.githooks`
- 將「我接下來可能會 push」與「我現在要親自執行 `git push`」混為一談，導致過早改動 hook 設定
- 從命名慣例或主觀猜測推斷工作樹的來源分支
- 在目前 checkout 不是連結工作樹時仍顯示工作樹 rebase 或推送選單
- 將 `git worktree list` 中的主要 checkout 分支視為連結工作樹來源分支的證明
- 在使用者尚未選擇 rebase 路徑前，以「它來自哪個分支？」替換工作樹三選項問題

## 範例

好的範例：

- `ci(git): enforce quick scoped tests before push`
- `fix(logs): clean up old rotated files automatically`
- `feat(errc): add stable error code constants for callers`

不好的範例：

- `fix: improve writer rotation handling`
- `feat: add new constants`
- `docs: update README`
- `fix: update writer.go and rotate.go`
