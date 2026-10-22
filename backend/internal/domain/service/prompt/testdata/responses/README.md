# 解析回帰用の回答サンプル（11.7.2）

外部AIの回答を貼り戻したときの解析を回帰テストするためのサンプル。
`<use_case>/<template_version>/*.txt` に置き、期待する parse_result と取得件数を `expected.json` に記録する。
`expected.json` に載っていない `.txt` があるとテストは失敗する（記録漏れを防ぐ）。

現在の `*_style*.txt` などは、ChatGPT / Gemini の典型的な出力の癖（前置き・Markdown 装飾・全角数字・後書き）を再現した**合成サンプル**。
実際の回答は `chatgpt_real_<n>.txt` / `gemini_real_<n>.txt` として追加する（11.7.2 PR-005：新バージョンごとに各サービス 3 件以上）。

## 実回答の集め方

プロンプトは共通テストデータ（11.0）から生成済みのゴールデンファイルを使う。実在の個人情報は入らない。

| use_case / バージョン | 外部AIに貼るプロンプト | 保存先 |
| --- | --- | --- |
| `qa_generate` / `qa-v1` | `../golden/qa-v1.txt` | `qa_generate/qa-v1/` |
| `answer_review` / `review-v1` | `../golden/review-v1.txt` | `answer_review/review-v1/` |
| `research_summarize` / `research-v1` | `../golden/research-v1.txt` | `research_summarize/research-v1/` |

1. ゴールデンファイルの中身をそのまま ChatGPT（または Gemini）に貼って送る
2. 回答の「コピー」ボタンで全文をコピーし、保存先に `chatgpt_real_1.txt` のような名前で保存する（前置きや後書きも消さない）
3. 期待値を記録する（`backend` ディレクトリで実行）

```bash
go test ./internal/domain/service/prompt -run TestParseFixtures -update
git diff internal/domain/service/prompt/testdata/responses/expected.json
```

4. 追加された行の `result` と件数が、回答を目で見た内容と合っているか確認する。合っていなければ、解析（`parse.go`）の不具合として直す
