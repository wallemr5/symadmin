# 调试
## 配置hosts
```shell
# 集群kube-apiserver
10.13.135.251 cls-89a4hpb3.ccs.tencent-cloud.com
10.13.133.9   cls-nkp08j5d.ccs.tencent-cloud.com
10.13.135.12  cls-7xq1bq9f.ccs.tencent-cloud.com

# helm chart 仓库
10.13.135.250 chartmuseum.dmall.com
```

## goland参数配置
```shell
# sym-controller 需要配置主集群tcc-bj5-dks-monit-01
sym-controller controller --enable-master --kubeconfig=./manifests/kubeconfig_TCC_BJ5_DKS_MONIT_01.yaml -v 4
# sym-operator   可以配置worker集群 tcc-bj4-dks-test-01/tcc-bj5-dks-test-01
sym-controller controller --enable-worker --kubeconfig=./manifests/kubeconfig_TCC_BJ4_DKS_TEST_01.yaml -v 4
# sym-api        需要配置主集群tcc-bj5-dks-monit-01
sym-api api --kubeconfig=./manifests/kubeconfig_TCC_BJ5_DKS_MONIT_01.yaml -v 4
```

# 发布

```shell

```


# 其他

## 初始化 pre-commit 钩子

可选择安装 [pre-commit](https://pre-commit.com/)，在每次 `git commit` 时对提交的文件自动执行常见的 `lint` 检查，避免低级错误：

```cmd
$ brew install pre-commit // 通过 Homebrew 安装
$ pip install pre-commit // 通过 Python 安装
$ pre-commit install  // 安装 pre-commit 钩子
```

检查工具需要安装以下依赖：

```sh
GO111MODULE=on CGO_ENABLED=0 go get -v -trimpath -ldflags '-s -w' github.com/golangci/golangci-lint/cmd/golangci-lint
go get -v -u github.com/BurntSushi/toml/cmd/tomlv
go get -v github.com/go-lintpack/lintpack/...
```

参考：[pre-commit-golang](https://github.com/dnephin/pre-commit-golang)

`golangci-lint` 检查项可通过 [.golangci.yml](./.golangci.yml) 配置。

关闭 `pre-commit` 钩子：

```cmd
$ pre-commit uninstall
```

*注：通过 UI 界面进行 `git` 操作的话会被隐藏至后台，无法查看。建议通过命令执行 `git commit`*
