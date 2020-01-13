package apiManager

// GetClusterDesc describing api /api/cluster/:name
var GetClusterDesc = `
Get cluster status:<br/>
name: url param,the unique cluster name and all <br/>
e.g. <br/>
<a href="/api/cluster/all">get all cluster status</a><br/>
`

// GetPodDesc ...
var GetPodDesc = `
Get all the pods which an app belongs: <br/>
name: url param,the unique cluster name and all <br/>
appName: url param,the unique app name. <br>
e.g. <br/>
`

// GetNodeProjectDesc ...
var GetNodeProjectDesc = `
Get all pods on a node: <br/>
name: url param,the unique cluster name and all <br/>
ip: url param,the unique node ip. <br/>
e.g. <br/>
`

// DeletePodByGroupDesc ...
var DeletePodByGroupDesc = `
Delete all the pods which app label is blue/green: <br/>
name: url param,the unique cluster name and all. <br/>
appName: url param,the unique app name. <br/>
e.g. <br/>
`

// DeletePodByNameDesc ...
var DeletePodByNameDesc = `
Delete a pod with pod name: <br/>
name: the unique cluster name and all <br/>
appName: url param, the unique app name <br/>
podName: url param, the unique pod name <br/>
namespace: query string, the unique namespace name. <br/>
e.g. <br/>
`

// GetEndpointsDesc ...
var GetEndpointsDesc = `
Get all endpoint by name: <br/>
name: url param, the unique cluster name and all. <br/>
endpointName: url param, the unique endpoint name. <br/>
e.g. <br/>
`

// GetNodeInfoDesc ...
var GetNodeInfoDesc = `
Get node info by name: <br/>
name: url param,the unique cluster name and all. <br/>
nodeName: url param,the unique node name. <br/>
e.g. <br/>
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
e.g. <br/>
`

// GetServicesDesc ...
var GetServicesDesc = `
Get all services for a specific app in the cluster. <br/>
name: url param, the unique cluster name and all. <br/>
appName: url param, the unique app name. <br/>
e.g. <br/>
`

// GetDeploymentsDesc ...
var GetDeploymentsDesc = `
Get all deployments in assigned namespace. <br/>
name: url param, the unique cluster name and all. <br/>
namespace: query string, the unique namespace in a cluster. Default is 'default' namespace. <br/>
e.g. <br/>
`

// GetPodEventDesc ...
var GetPodEventDesc = `
Get a limited number of pod events. <br/>
name: url param, the unique cluster name and all. <br/>
appName: url param,the unique app name. <br>
namespace: query string, the unique namespace in a cluster. Default is 'default' namespace. <br/>
podName: url param, the unique pod name. <br/>
limit: query string, the limit number, default is 10. <br/>
e.g. <br/>
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
e.g. <br/>
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
e.g. <br/>
`
