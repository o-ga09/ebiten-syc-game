# じんせいGAME（仮）

山本彩「game」にちなんだ、非公式のファンメイドのブラウザゲームです。
ガチャで決まった才能・難易度・家庭環境のまま、左右タップで人生の段をどこまでも登ります。

- ゲーム本体：Go + [Ebitengine](https://ebitengine.org/)（wasm）
- サーバー：Go + [syumai/workers-go](https://github.com/syumai/workers-go)（Cloudflare Workers）+ D1
- 仕様：[docs/SPEC.md](docs/SPEC.md)

## 必要なもの

- Go 1.24 以上
- Node.js（wrangler 用）

## 開発

```sh
npm install
make test           # Go のテスト
make dev            # ローカル D1 のマイグレーション → wrangler dev（http://localhost:8787）
make local          # wrangler を使わない簡易サーバー（メモリ上のストア）
go run ./game       # デスクトップで起動（GAME_API=http://localhost:8787 で API に接続）
```

## デプロイ

```sh
npx wrangler d1 create jinsei-game   # 表示された database_id を wrangler.jsonc に書く
make deploy                          # リモート D1 のマイグレーション → デプロイ
```

## 操作

| 入力 | 動作 |
|---|---|
| 画面の左半分 / ← / A | 左上の段へ |
| 画面の右半分 / → / D | 右上の段へ |
| R / 右上のボタン | リセット再起動（3回目のプレイから） |
| Esc / 右上の II | ポーズ |

## 権利について

- 楽曲は Apple Music の公式埋め込みプレイヤーでのみ再生します。音源はホストしていません。
- 歌詞はゲーム内に使用していません（うたネットへのリンクのみ）。
- フォント：[hajimehoshi/bitmapfont](https://github.com/hajimehoshi/bitmapfont)（Apache 2.0 ほか、同梱フォントの各ライセンスに従います）。
