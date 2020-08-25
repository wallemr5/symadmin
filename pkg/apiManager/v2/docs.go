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
