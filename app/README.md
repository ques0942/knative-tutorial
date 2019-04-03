# README
knativeチュートリアル用のアプリ

## Setup
### 前提として
プロジェクトのFirestoreをnativeモードで有効化しておいてください

## 鍵の作成
サーバアプリケーションからGCPのAPIを使用するためにはサービスアカウントが必要です。
詳細を知りたい場合は[こちら](https://cloud.google.com/iam/docs/creating-managing-service-account-keys?hl=ja#iam-service-account-keys-create-gcloud)をご参照ください。

### サービスアカウントの確認
```shell-session
gcloud iam service-accounts list
```
少なくともデフォルトのアカウントがあるはずですが、
必要であれば権限を適切に設定したアカウントを作成してください。

### サービスアカウントキーの作成
```shell-session
gcloud iam service-accounts keys create service_account_key.json --iam-account ${iam-account}
```
iam-accountは「サービスアカウントの確認」で表示されたもの(メールアドレス形式の値が表示されているはずです)から
任意のアカウントを使用してください。
コマンドを実行すると、「service_account_key.json」というファイルが作成されます。


## API


## Docker
```shell-session
docker run \
    --rm \
    -e PROJECT_ID=${project_id} \
    -e GOOGLE_APPLICATION_CREDENTIALS=/cred/key.json
    -v $(pwd)/service_account_key.json:/cred/key.json
    
    
```
