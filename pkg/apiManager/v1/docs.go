package v1

// GetClusterDesc describing api /api/cluster/:name
var GetClusterDesc = `
Get cluster status:<br/>
name: url param,the unique cluster name and all <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/all">/api/cluster/all</a><br/>
`

// GetClusterResourceDesc ...
var GetClusterResourceDesc = `
Get cluster resources, include pods,services and deployments.<br/>
name: url param,the unique cluster name and all <br/>
namespace: url param, namespace name <br/>
appName: url param,the unique app name. <br/>
group: query string, the unique group name. <br/>
zone : query string, the unique zone name. <br/>
ldcLabel: query string, the unique ldcLabel name. <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/all/namespace/default/app/bbcc/resource?group=blue">/api/cluster/all/namespace/default/app/bbcc/resource?group=blue</a><br/>
`

// GetPodByNameDesc ...
var GetPodByNameDesc = `
Get all the pods which an app belongs: <br/>
name: url param,the unique cluster name and all <br/>
podName: url param,the unique pod name. <br/>
namespace: url param, the unique namespace name. <br/>
group: query string, the unique group name. <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/all/appPod/bbcc?group=blue">/api/cluster/all/appPod/bbcc?group=blue</a><br/>
`

// GetPodByLabelsDesc ...
var GetPodByLabelsDesc = `
Get all the pods which an app belongs: <br/>
name: url param,the unique cluster name and all <br/>
appName: query param,the unique app name. <br/>
group: query string, the unique group name. <br/>
zone : query string, the unique zone name. <br/>
ldcLabel: query string, the unique ldcLabel name. <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/all/appPod/labels?appName=bbcc&group=blue">/api/cluster/all/appPod/labels?appName=bbcc&group=blue</a><br/>
`

// DeletePodByGroupDesc ...
var DeletePodByGroupDesc = `
Delete all the pods which app label is blue/green: <br/>
name: url param,the unique cluster name and all. <br/>
namespace: url param, namespace name <br/>
appName: url param,the unique app name. <br/>
group: query string, the unique group label. <br/>
zone : query string, the unique zone name. <br/>
ldcLabel: query string, the unique ldcLabel name. <br/>
<br/>
e.g. <br/>
<a>/api/cluster/tcc-bj5-dks-monit-01/namespace/default/app/bbcc?&group=blue</a><br/>
`

// DeletePodByNameDesc ...
var DeletePodByNameDesc = `
Delete a pod with pod name: <br/>
name: the unique cluster name and all <br/>
namespace: url param, namespace name <br/>
podName: url param, the unique pod name <br/>
<br/>
e.g. <br/>
<a>/api/cluster/tcc-bj5-dks-monit-01/namespace/default/pod/bbcc-xx-xx</a><br/>
`

// GetEndpointsDesc ...
var GetEndpointsDesc = `
Get all endpoint by name: <br/>
name: url param, the unique cluster name and all. <br/>
appName: url param,the unique app name. <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/all/endpoints/bbcc?group=blue">/api/cluster/all/endpoints/bbcc?group=blue</a><br/>
`

// GetTerminalDesc ...
var GetTerminalDesc = `
Use websocket to connect to the container in the pod, you can choose whether to output<br/>
stdout with tty, or you can disconnect after executing a command once. <br/>
name: url param, the unique cluster name and all. <br/>
namespace: query string, the unique namespace in a cluster. Default is 'default' namespace. <br/>
pod: query string, the unique pod name. <br/>
container: query string, the unique container name. <br/>
tty: query string, this parameter determines whether to output as tty. Default is true. <br/>
isStdin: query string, this parameter determines whether open stdin. Default is true. <br/>
isStdout: query string, this parameter determines whether open stdout. Default is true. <br/>
isStderr: query string, this parameter determines whether open stderr. Default is true. <br/>
once: query string, this parameter determines whether to execute a command and exit. <br/>
cmd: query string, commands executed in the container. <br/>
<br/>
e.g. <br/>
<a>ws://localhost:8080/api/cluster/tcc-bj5-dks-monit-01/terminal?namespace=default&pod=bbcc-xx-xx&container=bbcc</a><br/>
`

// ExecOnceWithHTTPDesc ...
var ExecOnceWithHTTPDesc = `
Use HTTP to executed commands in the container. <br/>
name: url param, the unique cluster name and all. <br/>
namespace: query string, the unique namespace in a cluster. Default is 'default' namespace. <br/>
pod: query string, the unique pod name. <br/>
container: query string, the unique container name. <br/>
tty: query string, this parameter determines whether to output as tty. Default is false. <br/>
cmd: query string, commands executed in the container. <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/tcc-bj5-dks-monit-01/exec?namespace=default&pod=bbcc-xx-xx&container=bbcc&tty=false&cmd=ls -a">/api/cluster/tcc-bj5-dks-monit-01/exec?namespace=default&pod=bbcc-xx-xx&container=bbcc&tty=false&cmd=ls -a</a><br/>
`

// GetServicesDesc ...
var GetServicesDesc = `
Get all services for a specific app in the cluster. <br/>
name: url param, the unique cluster name and all. <br/>
appName: url param, the unique app name. <br/>
group: query string, the unique group name. <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/all/namespace/dmall-inner/services/erp-dmall-test-wm-gw">/api/cluster/all/namespace/dmall-inner/services/erp-dmall-test-wm-gw</a><br/>
`

// GetDeploymentsDesc ...
var GetDeploymentsDesc = `
Get all deployments in assigned namespace. <br/>
name: url param, the unique cluster name and all. <br/>
namespace: url string, the unique namespace in a cluster.<br/>
appName: url string, the unique app name and all.<br/>
group: query string, the unique group name. <br/>
symZone: query string, the unique zone name. <br/>
ldcLabel: query string, the unique ldc name. <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/all/namespace/dmall-inner/deployments/all?symZone=gz01">/api/cluster/all/namespace/dmall-inner/deployments/all?symZone=gz01</a><br/>
`

// GetDeploymentInfoDesc ...
var GetDeploymentInfoDesc = `
Get a deployment in assigned namespace. <br/>
name: url param, the unique cluster name and all. <br/>
namespace: url string, the unique namespace in a cluster. Default is 'default' namespace. <br/>
deployName: url string, the unique name of deployment. <br/>
format: query string, default is yaml, return json response when the format value is not yaml. <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/tcc-gz01-bj5-test/namespace/dmall-inner/deployment/dm-cx-supplier-gz01a-blue">/api/cluster/tcc-gz01-bj5-test/namespace/dmall-inner/deployment/dm-cx-supplier-gz01a-blue</a><br/>
`

// GetServiceInfoDesc ...
var GetServiceInfoDesc = `
Get a service in assigned namespace. <br/>
name: url param, the unique cluster name and all. <br/>
namespace: url string, the unique namespace in a cluster. Default is 'default' namespace. <br/>
svcName: url string, the unique name of service. <br/>
format: query string, default is yaml, return json response when the format value is not yaml. <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/tcc-gz01-bj5-test/namespace/dmall-inner/service/erp-dmall-test-wm-gw-dmall-com">/api/cluster/tcc-gz01-bj5-test/namespace/dmall-inner/service/erp-dmall-test-wm-gw-dmall-com</a><br/>
`

// GetDeploymentsStatDesc ...
var GetDeploymentsStatDesc = `
Get all deployments statistics info in assigned namespace. <br/>
name: url param, the unique cluster name and all. <br/>
namespace: query string, the unique namespace in a cluster. Default is 'default' namespace. <br/>
appName: query string, the unique app name. <br/>
group: query string, the unique group name. <br/>
ldcLabel: query string, the unique ldc name. <br/>
zone: query string, the unique zone name. <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/all/deployment/stat?group=blue">/api/cluster/all/deployment/stat?group=blue</a><br/>
`

// GetPodEventDesc ...
var GetPodEventDesc = `
Get a limited number of pod events. <br/>
name: url param, the unique cluster name and all. <br/>
namespace: url param, namespace name <br/>
podName: url param, the unique pod name. <br/>
limit: query string, the limit number, default is 10. <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/all/namespace/default/pods/all/event">/api/cluster/all/namespace/default/pods/all/event</a><br/>
`

// GetWarningEventsDesc ...
var GetWarningEventsDesc = `
Get warning events of pods and advdeployment. <br/>
name: url param, the unique cluster name and all. <br/>
namespace: url param, namespace name <br/>
appName: query param, the unique app name. <br/>
group: query string, the unique group name. <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/all/namespace/default/events/warning?appName=aabb&group=blue">/api/cluster/all/namespace/default/events/warning?appName=aabb&group=blue</a><br/>
`

// GetFilesDesc ...
var GetFilesDesc = `
Get the file list of the specified directory. <br/>
name: query param, the unique cluster name and all. <br/>
namespace: query param, namespace name <br/>
podName: query param, the unique pod name. <br/>
container: query string, the unique container in a pod. <br/>
projectCode: query string, the unique project code. <br/>
appCode: query string, the unique app code. <br/>
appName: query string, the unique app name. <br/>

<br/>
e.g. <br/>
<a href=""></a>/api/pod/logfiles?name=aa&podName=bbcc&container=sym-api&projectCode=bbcc<br/>
`

// HandleLogsDesc ...
var HandleLogsDesc = `
Get stdout of a specific pod container. <br/>
name: url param, the unique cluster name and all. <br/>
namespace: url param, namespace name <br/>
podName: url param, the unique pod name. <br/>
container: query string, the unique container in a pod. <br/>
tail: query string, the log tail number, default is 1000. <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/tcc-bj5-dks-monit-01/namespace/default/pods/bbcc-xx-xx/logs?container=bbcc&tailLines=100">/api/cluster/tcc-bj5-dks-monit-01/namespace/default/pods/bbcc-xx-xx/logs?container=bbcc&tailLines=100</a><br/>
`

// HandleFileLogsDesc ...
var HandleFileLogsDesc = `
Get log files for a specific pod container. <br/>
name: url param, the unique cluster name and all. <br/>
namespace: url param, namespace name <br/>
podName: url param, the unique pod name. <br/>
container: query string, the unique container in a pod. <br/>
tailLines: query string, the log tail number, default is 100. <br/>
filepath: query string, the log file path in a container. <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/tcc-bj5-dks-monit-01/namespace/default/pods/bbcc-xx-xx/logs/file?container=bbcc&filepath=thanos.shipper.json">/api/cluster/tcc-bj5-dks-monit-01/namespace/default/pods/bbcc-xx-xx/logs/file?container=bbcc&filepath=xx/xx.log</a><br/>
`

// GetHelmReleasesDesc ...
var GetHelmReleasesDesc = `
Get helm releases for a specific app name. </br>
name: url param, the unique cluster name and all. <br/>
appName: url param, the unique app name and all. <br/>
group: query string, the unique group name.
<br/>
e.g.<br/>
<a href="/api/cluster/tcc-bj5-dks-test-01/appPod/aabb-9000/helm?group=blue">/api/cluster/tcc-bj5-dks-test-01/appPod/aabb-9000/helm?group=blue</a><br/>
`

// GetHelmReleaseInfoDesc ...
var GetHelmReleaseInfoDesc = `
Get helm releases for a specific app name. </br>
name: url param, the unique cluster name and all. <br/>
releaseName: url param, the unique release name. <br/>
<br/>
e.g.<br/>
<a href="/api/cluster/tcc-bj5-dks-test-01/helm/aabb-9000-rz01a-blue">/api/cluster/tcc-bj5-dks-test-01/helm/aabb-9000-rz01a-blue</a><br/>
`
