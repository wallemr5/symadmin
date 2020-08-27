package v2

// GetClusterDesc describing api /api/cluster/:name
var GetClusterDesc = `
Get cluster status:<br/>
name: url param,the unique cluster name and all <br/>
<br/>
e.g. <br/>
<a href="/api/cluster/all">/api/cluster/all</a><br/>
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
