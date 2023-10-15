# psh (parallel shell)

## 概要

- 複数のEC2インスタンスに対して並列でssh/scpコマンドを実行するためのツール
- サーバの台数が多い場合に短時間での処理が可能
- サーバの対象はサーバに付与しているタグで指定する
  - タグはカンマ区切りで複数指定可能
- scpの場合、 `-z` オプションを付与することで、scp後にファイルの展開を行う
  - 下記の拡張子をサポート
  - .tar / .tar.gz / .gz / .zip
- spcの際、 `-c` オプションを付与することでディレクトリが存在しない場合でも、作成することが可能
- 処理実行前に実行コマンドのプレビューが可能
  - `-y` オプションを付与することでプレビューのスキップが可能
- `-t` オプションを指定しない場合、describe-instancesで表示されるすべての起動中のインスタンスに対してコマンドが実行される

## 前提

- AWS環境での動作を想定している
  - AWSのアクセスキー/シークレットキーが設定されている、もしくはIAM Roleが設定されていること
- pshを実行するサーバには、対象のEC2インスタンスをを抽出するために下記の権限が必要になる

IAM Policy

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": "ec2:describeInstances",
            "Resource": "*"
        }
    ]
}
```

- 対象のEC2インスタンスにはsshでのアクセスが可能であること
- `-z` オプションの使用する場合、リモートインスタンス側に展開用のコマンドがインストールされている必要がある
  - .tar / .tar.gz: tar
  - .gz: gunzip
  - .zip: unzip

## インストール方法

```sh
version="v0.1.0"
arch="darwin-arm64"

curl -L https://github.com/yasuyuki0321/psh/releases/download/${version}/psh-${arch}.tar.gz | tar zxvf -
chmod 755 psh-${arch}

※ 必要に応じてリンクを作成したり、/bin等、PATHの通っているディレクトリに配置する
ln -s ./psh-${arch} ./psh
mv ./psh-${arch} /bin/
```

## 使用方法

### ssh

```text
execute SSH command across multiple targets

Usage:
  psh ssh [flags]

Flags:
  -c, --command string       command to execute via SSH
  -h, --help                 help for ssh
  -i, --ip-type string       select IP type: public or private (default "private")
  -k, --private-key string   path to private key (default "~/.ssh/id_rsa")
  -t, --tags string          comma-separated list of tag key=value pairs Example: Key1=Value1,Key2=Value2
  -u, --user string          username for SSH (default "ec2-user")
  -y, --yes                  skip the preview and execute the command directly
```

### scp

```text
execute scp operations across multiple targets

Usage:
  psh scp [flags]

Flags:
  -c, --create-dir           create the directory if it doesn't exist
  -z, --decompress           decompress the file after SCP
  -d, --dest string          dest file
  -h, --help                 help for scp
  -i, --ip-type string       select IP type: public or private (default "private")
  -m, --permission string    permission (default "644")
  -k, --private-key string   path to private key (default "~/.ssh/id_rsa")
  -s, --source string        source file
  -t, --tags string          comma-separated list of tag key=value pairs. Example: Key1=Value1,Key2=Value2
  -u, --user string          username to execute SCP command (default "ec2-user")
  -y, --yes                  skip the preview and execute the SCP directly
```

## コマンドの実行例

### ssh

```text
$ ./psh ssh -t Name=test,ssh=true -k ~/.ssh/yasuyuki0321-rsa.pem -i public -u ec2-user -c "uname -n"
Targets:
Name: test / ID: i-0a9ad44aa54f06a79 / IP: 57.180.27.233
Name: test / ID: i-068112822e1c8efd8 / IP: 13.112.120.101

Command: uname -n

Do you want to continue? [y/N]: y
----------
Time: 2023-10-15 10:46:59
ID: i-0a9ad44aa54f06a79
Name: test
IP: 57.180.27.233
Command: uname -n
----------
ip-10-0-0-39.ap-northeast-1.compute.internal

----------
Time: 2023-10-15 10:46:59
ID: i-068112822e1c8efd8
Name: test
IP: 13.112.120.101
Command: uname -n
----------
ip-10-0-0-113.ap-northeast-1.compute.internal

finish
```

### scp

```text
./psh scp -t Name=test,ssh=true -k ~/.ssh/yasuyuki0321-rsa.pem -i public -u ec2-user -s ./test.txt -d ./test.txt -m 0644
Targets:
Name: test / ID: i-068112822e1c8efd8 / IP: 13.112.120.101
Name: test / ID: i-0a9ad44aa54f06a79 / IP: 57.180.27.233

Source: ./test.txt
Destination: ./test.txt
Permission: 0644

Do you want to continue? [y/N]: y
----------
Time: 2023-10-15 10:48:15
ID: i-0a9ad44aa54f06a79
Name: test
IP: 57.180.27.233
Source: ./test.txt
Destination: ./test.txt
Permission: 0644
----------
-rw-r--r-- 1 ec2-user ec2-user 0 Oct 15 01:48 ./test.txt

----------
Time: 2023-10-15 10:48:15
ID: i-068112822e1c8efd8
Name: test
IP: 13.112.120.101
Source: ./test.txt
Destination: ./test.txt
Permission: 0644
----------
-rw-r--r-- 1 ec2-user ec2-user 0 Oct 15 01:48 ./test.txt

finish
```
