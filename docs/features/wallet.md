# Wallet（個人間精算台帳）仕様

## この文書の位置づけ

この文書は、実装前の Wallet v1 仕様を定義するためのものです。

- システムは現金や預かり金を管理しない
- 銀行振込、現金手渡し、送金アプリなどの実際の支払いはユーザー同士がシステム外で行う
- API は「誰が誰にいくら払うべきか」を仮想的に記録・可視化する
- 仮想通貨、ブロックチェーン、オンチェーン資産は扱わない

目的は、個人間の現金のやり取りを仮想化し、送金、請求、未清算のやり取り、清算済みのやり取りを一元的に追えるようにすることです。

## 概要

Wallet は以下を提供します。

- 自分の収支サマリ表示
- やり取り履歴表示
- ユーザー間の送金記録
- ユーザー間の請求リクエスト
- 選択したやり取りの清算

この機能で扱う金額は、サーバー上の台帳金額です。実際の支払いを代行するのではなく、「あとでいくら払うべきか」「どのやり取りをどの支払いで清算したか」を記録します。

注意:

- システムへの入金、システムからの出金はない
- 残高は「システムに預けているお金」ではなく、ユーザー間の債権債務を要約した数字
- 清算は、選択したやり取りに対して行う。全残高を一括で自動相殺する仕様ではない

## 通貨と金額の扱い

- 通貨は初期版では `JPY` 固定
- 金額は整数の円で扱う
- 小数は扱わない
- API 上の `amount` は 1 以上の整数

例:

- `1000` は 1000 円を意味する
- `0.5` や `"1000"` は不正

## 中心となる考え方

### net balance

各ユーザーについて、未清算のやり取りだけを集計した差額です。

- 正の値: そのユーザーは他人から受け取る側
- 負の値: そのユーザーは他人へ支払う側

この値は参考表示用です。清算は常に個別のやり取りを選択して行います。

### transfer

あるユーザーが、別のユーザーへ「自分が支払う側」として記録するやり取りです。

### request

あるユーザーが、別のユーザーへ「支払ってほしい」と依頼するやり取りです。

請求リクエストは、相手が将来支払うべき候補として積まれます。初期版では、請求リクエスト作成時点で相手承認は不要とします。

### settlement

実際の支払いがシステム外で行われた後、その支払いで清算した対象やり取りを選択して記録する行為です。

清算は次の 2 段階で扱います。

- 作成: 支払いが行われたはずだ、という申告を登録する
- 確認: 相手が「実際に支払われた」と確認し、初めて対象イベントへ反映する

1 つの清算で複数の送金・請求リクエストをまとめて清算できます。

## 用語

### wallet event

送金または請求リクエストを表す基本単位です。

### open event

まだ完全には清算されていないやり取りです。

### settled event

全額が清算済みになったやり取りです。

### settlement allocation

1 回の清算で、どのやり取りにいくら充当したかを表す明細です。

### pending settlement

まだ相手確認が終わっていない清算です。対象イベントには仮に紐づきますが、この時点では未清算残額を減らしません。

## データモデル

初期版では、少なくとも次の概念を持つ前提とします。

### wallet_accounts テーブル

| カラム | 型 | 説明 |
| --- | --- | --- |
| `user_id` | uuid | users.id。主キー |
| `currency` | text | 初期版は `JPY` 固定 |
| `net_balance` | bigint | 未清算イベントから計算した差額のキャッシュ |
| `frozen` | boolean | 凍結中か |
| `created_at` | timestamptz | 作成日時 |
| `updated_at` | timestamptz | 更新日時 |

制約:

- `currency = 'JPY'`
- 1ユーザーにつき1口座

補足:

- `net_balance` は高速参照用の集計値であり、真の状態はイベントと清算明細から再計算できるようにする

### wallet_events テーブル

| カラム | 型 | 説明 |
| --- | --- | --- |
| `id` | uuid | 主キー |
| `kind` | text | `transfer` または `request` |
| `from_user_id` | uuid | 支払う側のユーザー |
| `to_user_id` | uuid | 受け取る側のユーザー |
| `amount` | bigint | 元金額 |
| `settled_amount` | bigint | 清算済み合計 |
| `status` | text | `open` / `partially_settled` / `settled` / `canceled` |
| `note` | text \/ null | メモ |
| `created_by_user_id` | uuid | 作成者 |
| `created_at` | timestamptz | 作成日時 |
| `updated_at` | timestamptz | 更新日時 |

制約:

- `kind in ('transfer', 'request')`
- `amount > 0`
- `settled_amount >= 0`
- `settled_amount <= amount`
- `from_user_id != to_user_id`
- `status in ('open', 'partially_settled', 'settled', 'canceled')`

意味:

- `from_user_id` は支払う義務を持つ側
- `to_user_id` は受け取る権利を持つ側
- `kind = 'transfer'` でも `kind = 'request'` でも、金銭方向の意味は同じ
- 違いは誰がそのやり取りを起票したか、どう見せるかにある

### wallet_settlements テーブル

| カラム | 型 | 説明 |
| --- | --- | --- |
| `id` | uuid | 主キー |
| `payer_user_id` | uuid | 実際に支払った側 |
| `payee_user_id` | uuid | 実際に受け取った側 |
| `amount` | bigint | 実支払い額 |
| `status` | text | `pending_confirmation` / `confirmed` / `rejected` / `canceled` |
| `note` | text \/ null | 支払いメモ |
| `created_by_user_id` | uuid | 清算記録を作成したユーザー |
| `confirmed_by_user_id` | uuid \/ null | 確認したユーザー |
| `confirmed_at` | timestamptz \/ null | 確認日時 |
| `rejected_by_user_id` | uuid \/ null | 却下したユーザー |
| `rejected_at` | timestamptz \/ null | 却下日時 |
| `created_at` | timestamptz | 作成日時 |

制約:

- `amount > 0`
- `payer_user_id != payee_user_id`
- `status in ('pending_confirmation', 'confirmed', 'rejected', 'canceled')`

### wallet_settlement_allocations テーブル

| カラム | 型 | 説明 |
| --- | --- | --- |
| `settlement_id` | uuid | wallet_settlements.id |
| `event_id` | uuid | 清算対象イベント |
| `amount` | bigint | このイベントへ充当した額 |
| `created_at` | timestamptz | 作成日時 |

制約:

- `amount > 0`
- 1イベントに対する累計充当額は、そのイベントの `amount` を超えない

## 業務ルール

### 1. システムは現金を持たない

- 送金記録や請求リクエストは、実際の現金受け渡しそのものではない
- 清算が記録されても、システム内で資金移動は発生しない
- あくまで「この支払いでこの債務を消した」という台帳更新だけを行う

### 2. 金銭方向は常に `from_user_id -> to_user_id`

- `from_user_id` は支払う側
- `to_user_id` は受け取る側
- 送金と請求リクエストで表現の違いはあっても、最終的な債権債務の向きはこの 2 つで表す

### 3. 清算は選択したやり取りに対してのみ行う

- 清算リクエストには、対象イベント一覧を明示的に含める
- 対象に含まれていない open event は清算されない
- 集計残高だけを見て自動的に他イベントへ充当してはいけない

### 4. 清算は相手確認後に確定する

- `POST /wallet/settlements` の時点では `pending_confirmation` を作成するだけ
- 対象イベントの `settled_amount` は、`confirmed` になるまで更新しない
- 清算の確認は、作成者ではない当事者が行う
- 確認された時点でのみ、対象イベントへ allocation が反映される

### 5. 部分清算を許可する

- 1つのイベントは複数回に分けて清算できる
- 清算額が残額未満なら `partially_settled`
- 残額が 0 になった時点で `settled`

### 6. 未確認清算の二重充当を防ぐ

- `pending_confirmation` の清算に含まれている allocation も、対象イベントの仮予約として扱う
- あるイベントについて、`settled_amount + pending_confirmation allocations の合計` は `amount` を超えてはならない
- これにより、同じ未清算残額に対して複数の清算を重ねて登録することを防ぐ

### 7. 清算対象は同一の支払い方向に限る

1 回の清算で選べるイベントは、すべて次を満たす必要があります。

- `from_user_id` が同じ
- `to_user_id` が同じ
- `status` が `open` または `partially_settled`

理由:

- 1 回の現実支払いは 1 人から 1 人への支払いとして記録するため

### 8. 凍結中の口座は新規操作不可

`frozen = true` の口座では次を禁止します。

- 送金作成
- 請求リクエスト作成
- 清算作成
- 清算確認

閲覧は可能です。

### 9. 取消は論理状態で扱う

- 未清算イベントのみ `canceled` にできる
- `pending_confirmation` の清算に含まれているイベントは、清算が `rejected` または `canceled` になるまで取り消し不可
- 一部でも清算済みのイベントは直接取り消さず、必要なら逆方向の新規イベントで調整する

### 10. 確認前の清算はキャンセルできる

- `pending_confirmation` の清算のみ `canceled` にできる
- 清算キャンセルは、清算作成者のみ実行できる
- `canceled` になった清算はイベントへ反映されない
- キャンセル後、対象イベントに対する allocation の仮予約は解放される

## net balance の計算

各ユーザーについて、未清算残額を次で集計します。

- `incoming_open_amount = sum(未清算残額 where to_user_id = user)`
- `outgoing_open_amount = sum(未清算残額 where from_user_id = user)`
- `net_balance = incoming_open_amount - outgoing_open_amount`

ここで未清算残額は `amount - settled_amount` です。

補足:

- `pending_confirmation` の清算は `net_balance` を変えない
- ただし UI 上は「確認待ちの清算額」を別表示してよい

例:

- A -> B に 1000 円の open event がある
  - A の `net_balance = -1000`
  - B の `net_balance = 1000`

## 認証・認可

### 利用者向け API

- 認証は以下のどちらか
  - `Authorization: Bearer <JWT>`
  - Cookie（既存 Auth）
- 自分が関係するやり取りだけを参照できる
- 自分が `from_user_id` または `to_user_id` に含まれる相手との清算のみ作成できる
- 清算確認は、対象清算の当事者であり、かつ作成者ではない側のみ実行できる
- 清算キャンセルは、対象清算の作成者のみ実行できる

### イベント作成権限

- `transfer` は支払う側本人が作成することを原則とする
- `request` は受け取る側本人が作成することを原則とする

これにより、誰がどの立場で起票したかが明確になります。

## エンドポイント

Base: `http://localhost:3000`

### GET /wallet/me

自分の Wallet 概要を返します。

- 成功: `200`

レスポンス:

```json
{
  "userId": "2f0a3e8b-7f3f-4c60-9f8b-1a2b3c4d5e6f",
  "currency": "JPY",
  "netBalance": -2500,
  "incomingOpenAmount": 3000,
  "outgoingOpenAmount": 5500,
  "openEventCount": 4,
  "frozen": false,
  "updatedAt": "2026-03-09T10:00:00.000Z"
}
```

代表的なエラー:

- `401 UNAUTHORIZED`
- `404 WALLET_NOT_FOUND`

### GET /wallet/me/events

自分が関係するやり取り一覧を新しい順で返します。

- 成功: `200`

クエリ:

- `status`: `open` / `partially_settled` / `settled` / `canceled`（任意）
- `kind`: `transfer` / `request`（任意）
- `counterpartyUserId`: 相手ユーザーIDで絞り込み（任意）
- `counterpartyDiscordId`: 相手の Discord ID で絞り込み（任意）
- `counterpartyUserId` と `counterpartyDiscordId` は同時指定不可
- `limit`: 1〜100、デフォルト 20
- `cursor`: 追加読み込み用のカーソル

レスポンス:

```json
{
  "items": [
    {
      "id": "af9c1b86-9e1e-4ba1-9f76-5c5c6d0c4b2b",
      "kind": "request",
      "fromUserId": "c1b93d2d-78d9-4f1c-91da-fd1c7d4c7a82",
      "toUserId": "2f0a3e8b-7f3f-4c60-9f8b-1a2b3c4d5e6f",
      "amount": 1500,
      "settledAmount": 500,
      "remainingAmount": 1000,
      "status": "partially_settled",
      "note": "ランチ代",
      "createdByUserId": "2f0a3e8b-7f3f-4c60-9f8b-1a2b3c4d5e6f",
      "createdAt": "2026-03-09T09:30:00.000Z",
      "updatedAt": "2026-03-09T10:00:00.000Z"
    }
  ],
  "nextCursor": "eyJjcmVhdGVkQXQiOiIyMDI2LTAzLTA5VDA5OjMwOjAwLjAwMFoifQ=="
}
```

### GET /wallet/me/settlements

自分が関係する清算一覧を新しい順で返します。

- 成功: `200`

クエリ:

- `status`: `pending_confirmation` / `confirmed` / `rejected` / `canceled`（任意）

レスポンス:

```json
{
  "items": [
    {
      "id": "7cb1e299-f314-4ca4-a877-0fc00c7a53f9",
      "payerUserId": "c1b93d2d-78d9-4f1c-91da-fd1c7d4c7a82",
      "payeeUserId": "2f0a3e8b-7f3f-4c60-9f8b-1a2b3c4d5e6f",
      "amount": 1500,
      "status": "pending_confirmation",
      "note": "3/9 精算",
      "createdByUserId": "c1b93d2d-78d9-4f1c-91da-fd1c7d4c7a82",
      "confirmedByUserId": null,
      "confirmedAt": null,
      "createdAt": "2026-03-09T10:10:00.000Z",
      "allocations": [
        {
          "eventId": "af9c1b86-9e1e-4ba1-9f76-5c5c6d0c4b2b",
          "amount": 1000
        },
        {
          "eventId": "733179df-4211-4119-a8ce-91199bc71d2e",
          "amount": 500
        }
      ]
    }
  ]
}
```

### POST /wallet/transfers

自分が支払うべきやり取りを記録します。

- 成功: `201`

リクエスト:

```json
{
  "toUserId": "c1b93d2d-78d9-4f1c-91da-fd1c7d4c7a82",
  "amount": 1500,
  "note": "立て替えてもらったランチ代"
}
```

または:

```json
{
  "toDiscordId": "123456789012345678",
  "amount": 1500,
  "note": "立て替えてもらったランチ代"
}
```

意味:

- 認証ユーザーが `from_user_id`
- `toUserId` または `toDiscordId` で指定した相手が `to_user_id`
- `toUserId` と `toDiscordId` は同時指定不可

レスポンス:

```json
{
  "id": "d8d19469-6ef7-46d3-8597-f8f144e2af8f",
  "kind": "transfer",
  "fromUserId": "2f0a3e8b-7f3f-4c60-9f8b-1a2b3c4d5e6f",
  "toUserId": "c1b93d2d-78d9-4f1c-91da-fd1c7d4c7a82",
  "amount": 1500,
  "settledAmount": 0,
  "remainingAmount": 1500,
  "status": "open",
  "note": "立て替えてもらったランチ代",
  "createdByUserId": "2f0a3e8b-7f3f-4c60-9f8b-1a2b3c4d5e6f",
  "createdAt": "2026-03-09T09:30:00.000Z",
  "updatedAt": "2026-03-09T09:30:00.000Z"
}
```

代表的なエラー:

- `400 INVALID_REQUEST`
- `400 INVALID_AMOUNT`
- `400 SELF_TRANSFER_NOT_ALLOWED`
- `401 UNAUTHORIZED`
- `403 WALLET_FROZEN`
- `404 COUNTERPARTY_NOT_FOUND`

### POST /wallet/requests

相手へ請求リクエストを作成します。

- 成功: `201`

リクエスト:

```json
{
  "fromUserId": "c1b93d2d-78d9-4f1c-91da-fd1c7d4c7a82",
  "amount": 1500,
  "note": "ランチ代をお願いします"
}
```

または:

```json
{
  "fromDiscordId": "123456789012345678",
  "amount": 1500,
  "note": "ランチ代をお願いします"
}
```

意味:

- 認証ユーザーが `to_user_id`
- `fromUserId` または `fromDiscordId` で指定した相手が `from_user_id`
- `fromUserId` と `fromDiscordId` は同時指定不可

レスポンス:

```json
{
  "id": "6bd31191-1ae8-4dd5-94fe-7c0718a5b9d1",
  "kind": "request",
  "fromUserId": "c1b93d2d-78d9-4f1c-91da-fd1c7d4c7a82",
  "toUserId": "2f0a3e8b-7f3f-4c60-9f8b-1a2b3c4d5e6f",
  "amount": 1500,
  "settledAmount": 0,
  "remainingAmount": 1500,
  "status": "open",
  "note": "ランチ代をお願いします",
  "createdByUserId": "2f0a3e8b-7f3f-4c60-9f8b-1a2b3c4d5e6f",
  "createdAt": "2026-03-09T10:00:00.000Z",
  "updatedAt": "2026-03-09T10:00:00.000Z"
}
```

代表的なエラー:

- `400 INVALID_REQUEST`
- `400 INVALID_AMOUNT`
- `400 SELF_TRANSFER_NOT_ALLOWED`
- `401 UNAUTHORIZED`
- `403 WALLET_FROZEN`
- `404 COUNTERPARTY_NOT_FOUND`

### POST /wallet/settlements

実際の支払いが行われたという申告を登録し、選択したやり取りに対する確認待ち清算を作成します。

- 成功: `201`

リクエスト:

```json
{
  "payerUserId": "c1b93d2d-78d9-4f1c-91da-fd1c7d4c7a82",
  "payeeUserId": "2f0a3e8b-7f3f-4c60-9f8b-1a2b3c4d5e6f",
  "amount": 1500,
  "note": "3/9 精算",
  "allocations": [
    {
      "eventId": "af9c1b86-9e1e-4ba1-9f76-5c5c6d0c4b2b",
      "amount": 1000
    },
    {
      "eventId": "733179df-4211-4119-a8ce-91199bc71d2e",
      "amount": 500
    }
  ]
}
```

または:

```json
{
  "payerDiscordId": "123456789012345678",
  "payeeUserId": "2f0a3e8b-7f3f-4c60-9f8b-1a2b3c4d5e6f",
  "amount": 1500,
  "note": "3/9 精算",
  "allocations": [
    {
      "eventId": "af9c1b86-9e1e-4ba1-9f76-5c5c6d0c4b2b",
      "amount": 1000
    },
    {
      "eventId": "733179df-4211-4119-a8ce-91199bc71d2e",
      "amount": 500
    }
  ]
}
```

ルール:

- `payerUserId` / `payerDiscordId` のどちらか一方で支払者を指定する
- `payeeUserId` / `payeeDiscordId` のどちらか一方で受取者を指定する
- `allocations` の合計は `amount` と一致しなければならない
- 各 `eventId` は `payerUserId -> payeeUserId` の open event でなければならない
- 各 allocation はそのイベントの未清算残額から、他の `pending_confirmation` 清算で予約済みの額を引いた範囲を超えてはならない
- 認証ユーザーは `payerUserId` または `payeeUserId` のどちらかでなければならない
- レスポンス時点では `status = 'pending_confirmation'`
- この時点ではイベントの `settled_amount` はまだ増えない

レスポンス:

```json
{
  "id": "7cb1e299-f314-4ca4-a877-0fc00c7a53f9",
  "payerUserId": "c1b93d2d-78d9-4f1c-91da-fd1c7d4c7a82",
  "payeeUserId": "2f0a3e8b-7f3f-4c60-9f8b-1a2b3c4d5e6f",
  "amount": 1500,
  "status": "pending_confirmation",
  "note": "3/9 精算",
  "createdByUserId": "c1b93d2d-78d9-4f1c-91da-fd1c7d4c7a82",
  "confirmedByUserId": null,
  "confirmedAt": null,
  "createdAt": "2026-03-09T10:10:00.000Z",
  "allocations": [
    {
      "eventId": "af9c1b86-9e1e-4ba1-9f76-5c5c6d0c4b2b",
      "amount": 1000
    },
    {
      "eventId": "733179df-4211-4119-a8ce-91199bc71d2e",
      "amount": 500
    }
  ]
}
```

代表的なエラー:

- `400 INVALID_REQUEST`
- `400 INVALID_AMOUNT`
- `400 INVALID_SETTLEMENT_DIRECTION`
- `400 SETTLEMENT_AMOUNT_MISMATCH`
- `401 UNAUTHORIZED`
- `403 WALLET_FROZEN`
- `404 EVENT_NOT_FOUND`
- `409 EVENT_ALREADY_SETTLED`
- `409 PENDING_SETTLEMENT_CONFLICT`
- `409 SETTLEMENT_OVERFLOW`

### POST /wallet/settlements/:id/confirm

確認待ち清算について、実際に支払われたことを確認します。

- 成功: `200`

ルール:

- 対象清算の `status` は `pending_confirmation` でなければならない
- 認証ユーザーは `payerUserId` または `payeeUserId` の当事者でなければならない
- 認証ユーザーは `createdByUserId` と異なるユーザーでなければならない
- 確認成功時に、対象 allocations がイベントへ反映される

レスポンス:

```json
{
  "id": "7cb1e299-f314-4ca4-a877-0fc00c7a53f9",
  "status": "confirmed",
  "confirmedByUserId": "2f0a3e8b-7f3f-4c60-9f8b-1a2b3c4d5e6f",
  "confirmedAt": "2026-03-09T10:12:00.000Z"
}
```

代表的なエラー:

- `401 UNAUTHORIZED`
- `403 SETTLEMENT_CONFIRMATION_NOT_ALLOWED`
- `404 SETTLEMENT_NOT_FOUND`
- `409 SETTLEMENT_ALREADY_FINALIZED`

### POST /wallet/settlements/:id/reject

確認待ち清算について、実際の支払いを確認できないとして却下します。

- 成功: `200`

ルール:

- 対象清算の `status` は `pending_confirmation` でなければならない
- 認証ユーザーは `createdByUserId` と異なる当事者でなければならない
- 却下された清算はイベントへ反映されない
- 却下後、対象イベントは他の清算へ再度割り当て可能になる

代表的なエラー:

- `401 UNAUTHORIZED`
- `403 SETTLEMENT_CONFIRMATION_NOT_ALLOWED`
- `404 SETTLEMENT_NOT_FOUND`
- `409 SETTLEMENT_ALREADY_FINALIZED`

### POST /wallet/settlements/:id/cancel

確認待ち清算をキャンセルします。

- 成功: `200`

ルール:

- 対象清算の `status` は `pending_confirmation` でなければならない
- 認証ユーザーは `createdByUserId` と一致しなければならない
- キャンセルされた清算はイベントへ反映されない
- キャンセル後、対象イベントは他の清算へ再度割り当て可能になる

レスポンス:

```json
{
  "id": "7cb1e299-f314-4ca4-a877-0fc00c7a53f9",
  "status": "canceled"
}
```

代表的なエラー:

- `401 UNAUTHORIZED`
- `403 SETTLEMENT_CANCEL_NOT_ALLOWED`
- `404 SETTLEMENT_NOT_FOUND`
- `409 SETTLEMENT_ALREADY_FINALIZED`

### POST /wallet/events/:id/cancel

未清算イベントを取り消します。

- 成功: `200`

ルール:

- `status = 'open'` のイベントのみ取り消し可能
- `partially_settled` または `settled` のイベントは取り消し不可
- `pending_confirmation` の清算に含まれているイベントは取り消し不可
- 作成者または関係当事者のみ操作可能

## 業務フロー

### 送金フロー

1. 支払う側が `POST /wallet/transfers` を作成する
2. イベントは `open` になる
3. 後日、実際の支払い後に `POST /wallet/settlements` で対象イベントを選び、確認待ち清算を作成する
4. 必要なら作成者が `POST /wallet/settlements/:id/cancel` でキャンセルできる
5. 相手が `POST /wallet/settlements/:id/confirm` で確認すると清算が確定する

### 請求フロー

1. 受け取る側が `POST /wallet/requests` を作成する
2. イベントは `open` になる
3. 支払う側と受け取る側のどちらかが、実際の支払い後に `POST /wallet/settlements` を作成する
4. 必要なら作成者が確認前にキャンセルできる
5. もう片方の当事者が確認すると確定する

### 清算フロー

1. ユーザーが現実世界で支払う
2. その支払いで消したいやり取りを選ぶ
3. `POST /wallet/settlements` で allocation を明示して確認待ち清算を作成する
4. 作成者は確認前なら `POST /wallet/settlements/:id/cancel` で取り下げられる
5. 相手当事者が実際の支払いを確認する
6. 確認された場合のみ、対象イベントの `settled_amount` と `status` が更新される
7. 両者の `net_balance` が再計算または更新される

## 代表的なエラーコード

- `WALLET_NOT_FOUND`: 口座が未作成
- `INVALID_AMOUNT`: 金額が 1 以上の整数でない
- `SELF_TRANSFER_NOT_ALLOWED`: 自分自身とのやり取り
- `COUNTERPARTY_NOT_FOUND`: 相手ユーザーが存在しない
- `WALLET_FROZEN`: 凍結中のため操作不可
- `INVALID_SETTLEMENT_DIRECTION`: 清算対象の支払い方向が揃っていない
- `SETTLEMENT_AMOUNT_MISMATCH`: allocations 合計と清算額が一致しない
- `PENDING_SETTLEMENT_CONFLICT`: 他の確認待ち清算と競合している
- `EVENT_ALREADY_SETTLED`: 既に全額清算済み
- `SETTLEMENT_OVERFLOW`: 未清算残額を超えて清算しようとした
- `SETTLEMENT_CONFIRMATION_NOT_ALLOWED`: 清算確認権限がない
- `SETTLEMENT_CANCEL_NOT_ALLOWED`: 清算キャンセル権限がない
- `SETTLEMENT_ALREADY_FINALIZED`: 既に確認または却下済み

## スコープ外

初期版では次をスコープ外とします。

- システム預かり金
- 決済代行サービスとの連携
- 自動入金確認
- 複数通貨対応
- 為替計算
- 自動相殺ロジック
- 請求リクエストの承認ワークフロー