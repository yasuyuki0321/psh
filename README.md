# psh (parallel shell)

## 概要

- 複数のサーバに対して並列でssh/scpコマンドを実行するためのツールです。
- サーバの台数が多い場合に短時間での処理が可能になります。
- サーバの対象はサーバに付与しているタグで指定します。

## 前提

- AWS環境での動作を想定しています。

## 使用方法

### ssh

```sh
Execute SSH command across multiple targets

Usage:
  psh ssh [flags]

Flags:
  -c, --command string       Command to execute via SSH
  -h, --help                 help for ssh
  -t, --ip-type string       Select IP type: public or private (default "private")
  -p, --private-key string   Path to private key (default "~/.ssh/id_rsa")
  -k, --tag-key string       Tag key (default "Name")
  -v, --tag-value string     Tag value
  -u, --user string          Username for SSH (default "ec2-user")
```

### scp

```sh
A command to perform scp operations across multiple targets

Usage:
  psh scp [flags]

Flags:
  -d, --dest string          dest file
  -h, --help                 help for scp
  -t, --ip-type string       select IP type: public or private (default "private")
  -m, --permission string    permission
  -p, --private-key string   path to private key (default "~/.ssh/id_rsa")
  -s, --source string        source file
  -k, --tag-key string       tag key (default "Name")
  -v, --tag-value string     tag value
  -u, --user string          username to execute scp command (default "ec2-user")
```

## コマンドの実行例

### ssh

```sh
psh ssh -k Name -v test -p ~/.ssh/yasuyuki0321-rsa.pem -t public -u ec2-user -c "uname -n"                      
----------
Time: 2023-10-08 13:44:13
ID: i-0a9ad44aa54f06a79
IP: 13.112.249.250
Command: uname -n
----------
ip-10-0-0-39.ap-northeast-1.compute.internal

----------
Time: 2023-10-08 13:44:13
ID: i-068112822e1c8efd8
IP: 18.179.36.241
Command: uname -n
----------
ip-10-0-0-113.ap-northeast-1.compute.internal

finish
```

### scp

```sh
psh scp -k Name -v test -p ~/.ssh/yasuyuki0321-rsa.pem -t public -u ec2-user -s ./test.txt -d ./test.txt -m 0644
----------
Time: 2023-10-08 13:44:42
ID: i-0a9ad44aa54f06a79
IP: 13.112.249.250
Source: ./test.txt
Destination: ./test.txt
Permission: 0644
----------
-rw-r--r-- 1 ec2-user ec2-user 10 Oct  8 04:44 ./test.txt

----------
Time: 2023-10-08 13:44:42
ID: i-068112822e1c8efd8
IP: 18.179.36.241
Source: ./test.txt
Destination: ./test.txt
Permission: 0644
----------
-rw-r-xr-x 1 ec2-user ec2-user 10 Oct  8 04:44 ./test.txt
```
