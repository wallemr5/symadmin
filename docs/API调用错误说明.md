# API 调用错误说明

## 一、接口 HTTP 状态码说明

| 状态码                          | 说明                         |
|---------------------------------|------------------------------|
| 200 - OK                        | 接口返回正常                 |
| 204 - No Content                | 正确处理且没有数据返回       |
| 400 - Bad Request               | 请求错误，通常是因为缺少参数 |
| 401 - Unauthorized              | 缺少认证信息或认证信息错误   |
| 402 - Request Failed            | 参数正常但请求失败           |
| 403 - Forbidden                 | 禁止访问，缺少该资源权限     |
| 404 - Not Found                 | 请求的资源不存在             |
| 409 - Conflict                  | 请求冲突                     |
| 429 - Too Many Requests         | 过多的请求                   |
| 500,502,503,504 - Server Errors | 服务器错误                   |

## 二、接口错误类型说明

错误码使用四位数字，说明如下：

| 错误码 | 命名            | 说明                |
|--------|-----------------|---------------------|
|   1xxx | BaseError       | 基础错误类型        |
|   2xxx | PodError        | Pod 相关错误        |
|   3xxx | DeploymentError | Deployment 相关错误 |
|   4xxx | ServiceError    | Service 相关错误    |
|   5xxx | TerminalError   | Terminal 相关错误   |
|   6xxx | 预留            | 预留                |
|   7xxx | 预留            | 预留                |
|   8xxx | 预留            | 预留                |
|   9xxx | OtherError      | 其他类型错误          |

### 1. 基础错误

| 错误码 | 命名参考            | 说明                       |
|--------|---------------------|----------------------------|
|   1001 | ParamInvalidError   | 参数不合法                 |
|   1002 | RecordNotExistError | 记录不存在                 |

### 2. Pod 类型错误

| 错误码 | 命名参考         | 说明              |
|--------|------------------|-------------------|
|   2001 | GetPodError      | 获取 Pod 失败     |
|   2002 | GetPodEventError | 获取 Pod 事件失败 |

### 3. Deployment 类型错误

| 错误码 | 命名参考           | 说明                 |
|--------|--------------------|----------------------|
|   3001 | GetDeploymentError | 获取 Deployment 失败 |

### 4. Service 类型错误

| 错误码 | 命名参考           | 说明                 |
|--------|--------------------|----------------------|
|   4001 | GetServiceError| 获取 Service 失败 |

### 5. Terminal 类型错误

| 错误码 | 命名参考            | 说明                   |
|--------|---------------------|------------------------|
|   5001 | GetTerminalError    | 获取 Terminal 失败     |
|   5002 | WebsocketError      | Websocket 相关错误     |
|   5003 | RequestK8sExecError | 请求 K8s Exec 命令错误 |

### 9. 其他错误

| 错误码 | 命名参考         | 说明               |
|--------|------------------|--------------------|
|   9001 | GetClusterError  | 获取集群信息错误   |
|   9002 | GetEndpointError | 获取 Endpoint 错误 |
|   9003 | GetNodeError     | 获取 Node 错误     |

