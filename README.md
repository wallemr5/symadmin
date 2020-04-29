<img src="docs/symphony.png" alt="logo" width="250"/>

# 多点 DKS 混合云容器管理系统

[![pipeline status](http://gitlab.dmall.com/arch/sym-admin/badges/master/pipeline.svg)](http://gitlab.dmall.com/arch/sym-admin/commits/master)
[![coverage report](http://gitlab.dmall.com/arch/sym-admin/badges/master/coverage.svg)](http://gitlab.dmall.com/arch/sym-admin/commits/master)

**功能：**

- 支持混合云基础组件自动部署
- 兼容现有部署方式
- 支持多集群自动迁移
- 支持 k8s 原生 deployment、StatefulSet 等部署方式
- 支持多集群多活、单元化、蓝绿灰等多组高级部署方式
- 后期支持动态注入调度器亲和性调度策略、集群元数据环境

## 架构
![sym.png](docs/sym.png)

## 调试

### 配置 hosts

```shell
# 集群kube-apiserver
10.13.135.251 cls-89a4hpb3.ccs.tencent-cloud.com
10.13.133.9   cls-nkp08j5d.ccs.tencent-cloud.com
10.13.135.12  cls-7xq1bq9f.ccs.tencent-cloud.com

# helm chart 仓库
10.13.135.250 chartmuseum.dmall.com
```

### 调试 参数配置

```shell
# sym-controller 需要配置主集群tcc-bj5-dks-monit-01
sym-controller controller --enable-master --kubeconfig=./manifests/kubeconfig_TCC_BJ5_DKS_MONIT_01.yaml -v 4
# sym-operator   可以配置worker集群 tcc-bj4-dks-test-01/tcc-bj5-dks-test-01
sym-controller controller --enable-worker --kubeconfig=./manifests/kubeconfig_TCC_BJ4_DKS_TEST_01.yaml -v 4
# sym-api        需要配置主集群tcc-bj5-dks-monit-01
sym-api api --kubeconfig=./manifests/kubeconfig_TCC_BJ5_DKS_MONIT_01.yaml -v 4

# cluster 控制器调试
sym-controller controller --enable-cluster --enable-leader=false --kubeconfig=./manifests/kubeconfig_TCC_BJ5_DKS_MONIT_01.yaml -v 4
```

## 其他

### 初始化 pre-commit 钩子

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

_注：通过 UI 界面进行 `git` 操作的话会被隐藏至后台，无法查看。建议通过命令执行 `git commit`_

## 编译

### 编译`linux平台`二进制, 输出到*bin*目录

*生产环境二进制程序编译必须基于 `master` 分支。*

```shell
# 单独编译 api
make manager-api

# 单独编译 controller
make manager-controller

# api controller一起编译
make manager

# 使用 Docker 单独编译 api
make docker-build-api

# 使用 Docker 单独编译 controller
make docker-build-controller

# 使用 Docker 统一编译
make docker-build
```

### 打包 docker 镜像并推送

- `Dockerfile` 文件位于 `docker` 目录下，通过后缀区分
- 镜像地址修改 [Makefile](./Makefile) 中 `IMG_REG` 的值。
- 镜像打包必须基于 `master` 分支

```shell
# api
make docker-push-api

# controller
make docker-push-controller

# api & controller
make docker-push
```

## 发布

通过 `helm` 安装，`chart` 路径 `./chart/`

```shell
# controller master
make helm-master

# controller worker
make helm-worker

# controller master & worker
make helm-master-worker

# api
make helm-api
```

## Git 开发流程

项目维护两个一直延续的分支：

- master
- dev

其中 `master` 分支为主分支，随时处于预备生产状态。`dev` 为开发分支，用于合并其他辅助性分支（`feature`、`bugfix`、`doc` 等）。这两个分支都处于保护状态，禁止强推（`git push -f`），禁止 `Maintainer` 以下角色 `push` 和 `merge`。

### 发版流程

`release` 分支为发布做准备，用于修改版本号等元数据。发布期间 (还未上线) 的 Bug 修复可以提交到该分支上，但不允许新的 `feature` 提交（`feature` 须提交至 `dev` 分支，等待下次发布）。

基于 `dev` 创建 `release` 分支：

```shell
git checkout -b release-v* origin/dev
```

该分支测试无误后预备发布，将其合并到 `master` 和 `dev` 中:

```shell
# master
git checkout master
git merge origin/release-v*

# dev
git checkout dev
git merge origin/release-v*
```

基于 `master` 分支打 Tag 并同步至远程仓库 `origin`:
```shell
git tag -a v* -m "bumpversion v*"
git push origin v*
```

`release`分支在生产环境上线后不再维护，上线后修复问题须基于 `master` 分支创建 `hotfix` 分支进行修复，参见线上问题修复说明。

### 其他辅助分支约定

本地开发前，不同的修改最好基于 `dev`分支创建新的分支，尽量遵循以下命名规范:

- 新功能：`{姓名}/feature-{功能描述}`，如：`haidong/feature-add-pod-api`
- 问题修复：`{姓名}/bugfix-{问题描述}`，如：`haidong/bugfix-pod-api-return-error`
- 文档更新：`{姓名}/docs-{文档描述}`，如：`haidong/doc-update-readme`

其他如重构(`refactor`)、测试（`test`） 分支同理。

### 线上问题修复说明（Hotfix)

若生产环境出现问题需要修复，必须基于 `master` 分支创建 `hotfix` 分支进行修改：
```shell
git checkout -b haidong/hotfix-pod-api-error origin/master
```

完成修改后须推送至 GitLab 远程仓库并同时提交两个 PR 至 `master` 和 `dev` 分支，提醒管理员合并。

`Maintainer` 及以上角色可在本地进行合并（尽量在 GitLab 操作，PR 审核页面可以实时查看 CI 状态避免大部分错误）：
```shell
# 合并至 master
git checkout master
git merge origin/haidong/hotfix-***
git push origin master

# 合并至 dev
git checkout dev
git merge origin/haidong/hotfix-***
git push origin dev
```

合并后发布前需要在 `master` 分支打 Tag:
```shell
git tag -a v* -m "bumpversion v*"
git push origin v*
```

### 自动生成 CHANGELOG

*尽量规范 Commit 信息*

```shell
npm install -g conventional-changelog-cli
conventional-changelog -p angular -i CHANGELOG.md -s -p
```

详情参见：[conventional-changelog](https://github.com/conventional-changelog/conventional-changelog)

## 开发、测试集群整理

规范开发、测试环境集群，文件链接：[kubeconfig.yaml](./manifests/kubeconfig.yaml)

| kubeconfig 编码        | 环境      | 集群说明                   | 腾讯云编码   |      hosts ip | 备注 |
|------------------------|-----------|----------------------------|--------------|---------------|------|
| dev-tke-bj5-monit-01   | 开发/生产 | 老薛监控集群，可作为开发   | cls-89a4hpb3 | 10.13.135.251 |      |
| dev-tke-bj5-test-01    | 开发      | 组内开发集群               | cls-7xq1bq9f |  10.13.135.12 |      |
| cn-tke-bj5-test-01     | 测试      | 组内云原生北京测试集群     | cls-278pwqet | 10.16.247.131 |      |
| cn-tke-cd-test-01      | 测试      | 组内云原生成都测试集群     | cls-97rlivuj |  10.16.113.81 |      |
| test-tke-gz-bj5-bus-01 | 测试      | 业务测试单元化北京 GZ 集群 | cls-otdyiqyb |  10.16.247.78 |      |
| test-tke-rz-bj5-bus-01 | 测试      | 业务测试单元化北京 RZ 集群 | cls-h5f02nmb |  10.16.247.11 |      |
| test-tke-rz-cd-bus-01  | 测试      | 业务测试单元化成都 RZ 集群 | cls-3yclxq8t |  10.16.113.12 |      |
