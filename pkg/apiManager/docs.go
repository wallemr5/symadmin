package apiManager

// GetClusterDesc describing api /api/cluster/:name
var GetClusterDesc = `
Get cluster status:<br/>
name: url param,the unique cluster name and all <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/all">/api/cluster/all</a><br/>
`

// GetPodDesc ...
var GetPodDesc = `
Get all the pods which an app belongs: <br/>
name: url param,the unique cluster name and all <br/>
appName: url param,the unique app name. <br>
<br/>
e.g. <br/>
<a href="/api/cluster/all/appPod/helm">/api/cluster/all/appPod/helm</a><br/>
`

// GetNodeProjectDesc ...
var GetNodeProjectDesc = `
Get all pods on a node: <br/>
name: url param,the unique cluster name and all <br/>
ip: url param,the unique node ip. <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/all/nodeIp/10.13.135.17">/api/cluster/all/nodeIp/10.13.135.17</a><br/>
`

// DeletePodByGroupDesc ...
var DeletePodByGroupDesc = `
Delete all the pods which app label is blue/green: <br/>
name: url param,the unique cluster name and all. <br/>
appName: url param,the unique app name. <br/>
namespace: query string, the unique namespace name. <br/>
group: query string, the unique group label. <br/>
<br/>
e.g. <br/>
<a>/api/cluster/tcc-bj5-dks-monit-01/appPod/tidb-pd-0?namespace=tidb-admin&group=blue</a><br/>
`

// DeletePodByNameDesc ...
var DeletePodByNameDesc = `
Delete a pod with pod name: <br/>
name: the unique cluster name and all <br/>
appName: url param, the unique app name <br/>
podName: url param, the unique pod name <br/>
namespace: query string, the unique namespace name. <br/>
<br/>
e.g. <br/>
<a>/api/cluster/tcc-bj5-dks-monit-01/appPod/tidb-pd-0/pods/podname?namespace=tidb-admin</a><br/>
`

// GetEndpointsDesc ...
var GetEndpointsDesc = `
Get all endpoint by name: <br/>
name: url param, the unique cluster name and all. <br/>
endpointName: url param, the unique endpoint name. <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/all/endpointName/kubernetes">/api/cluster/all/endpointName/kubernetes</a><br/>
`

// GetNodeInfoDesc ...
var GetNodeInfoDesc = `
Get node info by name: <br/>
name: url param,the unique cluster name and all. <br/>
nodeName: url param,the unique node name. <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/all/nodeName/10.13.135.252">/api/cluster/all/nodeName/10.13.135.252</a><br/>
`

// GetTerminalDesc ...
var GetTerminalDesc = `
Use websocket to connect to the container in the pod, you can choose whether to output<br/>
stdout with tty, or you can disconnect after executing a command once. <br/>
name: url param, the unique cluster name and all. <br/>
namespace: query string, the unique namespace in a cluster. Default is 'default' namespace. <br/>
podName: query string, the unique pod name. <br/>
containerName: query string, the unique container name. <br/>
tty: query string, this parameter determines whether to output as tty. Default is true. <br/>
isStdin: query string, this parameter determines whether open stdin. Default is true. <br/>
isStdout: query string, this parameter determines whether open stdout. Default is true. <br/>
once: query string, this parameter determines whether to execute a command and exit. <br/>
cmd: query string, commands executed in the container. <br/>
<br/>
e.g. <br/>
<a>ws://localhost:8080/api/cluster/tcc-bj5-dks-monit-01/terminal?namespace=sym-admin&pod=prometheus-sym-apg-prometheus-0&container=prometheus</a><br/>
`

// GetServicesDesc ...
var GetServicesDesc = `
Get all services for a specific app in the cluster. <br/>
name: url param, the unique cluster name and all. <br/>
appName: url param, the unique app name. <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/all/service/appName">/api/cluster/all/service/appName</a><br/>
`

// GetDeploymentsDesc ...
var GetDeploymentsDesc = `
Get all deployments in assigned namespace. <br/>
name: url param, the unique cluster name and all. <br/>
namespace: query string, the unique namespace in a cluster. Default is 'default' namespace. <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/all/deployments">/api/cluster/all/deployments</a><br/>
`

// GetPodEventDesc ...
var GetPodEventDesc = `
Get a limited number of pod events. <br/>
name: url param, the unique cluster name and all. <br/>
appName: url param,the unique app name. <br>
namespace: query string, the unique namespace in a cluster. Default is 'default' namespace. <br/>
podName: url param, the unique pod name. <br/>
limit: query string, the limit number, default is 10. <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/all/appPod/all/pods/all/event">/api/cluster/all/appPod/all/pods/all/event</a><br/>
`

// GetFilesDesc ...
var GetFilesDesc = `
Get the file list of the specified directory. <br/>
name: url param, the unique cluster name and all. <br/>
appName: url param,the unique app name. <br>
podName: url param, the unique pod name. <br/>
namespace: query string, the unique namespace in a cluster. Default is 'default' namespace. <br/>
container: query string, the unique container in a pod. <br/>
path: query string, the log files directory path. <br/>

<br/>
e.g. <br/>
<a href="/api/cluster/tcc-bj5-dks-monit-01/appPod/aaa/pods/prometheus-sym-apg-prometheus-0/files?namespace=sym-admin&container=prometheus">/api/cluster/tcc-bj5-dks-monit-01/appPod/aaa/pods/prometheus-sym-apg-prometheus-0/files?namespace=sym-admin&container=prometheus</a><br/>
`

// HandleLogsDesc ...
var HandleLogsDesc = `
Get stdout of a specific pod container. <br/>
name: url param, the unique cluster name and all. <br/>
appName: url param,the unique app name. <br>
podName: url param, the unique pod name. <br/>
namespace: query string, the unique namespace in a cluster. Default is 'default' namespace. <br/>
container: query string, the unique container in a pod. <br/>
tailLines: query string, the log tail number, default is 100. <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/tcc-bj5-dks-monit-01/appPod/bbcc2/pods/prometheus-sym-apg-prometheus-0/logs?namespace=sym-admin&tailLines=100">/api/cluster/tcc-bj5-dks-monit-01/appPod/bbcc2/pods/prometheus-sym-apg-prometheus-0/logs?namespace=sym-admin&tailLines=100</a><br/>
`

// HandleFileLogsDesc ...
var HandleFileLogsDesc = `
Get log files for a specific pod container. <br/>
name: url param, the unique cluster name and all. <br/>
appName: url param, the unique app name. <br>
podName: url param, the unique pod name. <br/>
namespace: query string, the unique namespace in a cluster. Default is 'default' namespace. <br/>
container: query string, the unique container in a pod. <br/>
tailLines: query string, the log tail number, default is 100. <br/>
fileName: query string, the log file path in a container. <br/>
<br/>
e.g. <br/>
<a href=""></a><br/>
`
