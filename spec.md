# SLI/SLO ログ駆動ソリューション仕様（ALB ログ → CloudWatch カスタムメトリクス）

目的: **アプリ側の計測コンポーネントを導入せず**、ALB アクセスログから **パス（CUJ）単位のリクエスト系 SLI** を算出し、CloudWatch Application Signals の **SLO と複数ウィンドウ・複数バーンレートのアラーム**を運用可能にする。

このタスクでは、ALB アクセスログから CloudWatch カスタムメトリクスを生成する Lambda 関数を実装します。Lambda のランタイムは Go とする。

* ALB アクセスログ（S3）をもとに以下のカスタムメトリクスを投稿する
    * パスごとのリクエスト総数
    * パスごとのリクエスト成功数（HTTP 200〜499）
    * パスごとのレイテンシ（raw データ）

## メトリクス設計

* **Namespace**: `<Service name>/SLI`
* **Metrics**
  * `RequestsTotal` (Sum)
  * `RequestsGood` (Sum)
  * `Latency` (Average, p95/p99などはCloudWatch集計)
* **Dimensions**
  * `path`（正規化パス：例 `/orders/:id`）
* **良いリクエスト定義（デフォルト）**
  * HTTPステータス 200〜499
* **粒度**
  * 1分集計（Lambda 側で 1分バケットに集計し Sum を送信）
* **カーディナリティ管理**
  * ホワイトリスト制（監視対象 CUJ のみ）
  * `path` はプレースホルダー化で爆発抑制

## ルート正規化例

* `/orders/123` → `/orders/:id`
* `/users/550e8400-e29b-41d4-a716-446655440000` → `/users/:uuid`
* クエリ文字列除去、末尾スラッシュ正規化

## Lambda 集計・送信

* **入力**: ALB ログ 1 オブジェクト（TSV）
* **集計キー**: `{service, env, normalized_route, minute(bucketed)}`
* **集計値**: `count_total`, `count_good`, `latency_sum`, `latency_count`
* **送信**:

  * `PutMetricData` バッチ（最大1000データポイント/req, 1MB制限）
  * 1分バケット内は1回のみ送信（冪等化）
* **失敗時**: DLQ（SQS）へ送信し再処理可能


## 制約/トレードオフ

* 検知遅延: 5〜10分（ALBログ配信＋Lambda処理）
* 短時間スパイクへの感度は30分窓依存
* CUJ以外のパスはSLO対象外
