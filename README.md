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

