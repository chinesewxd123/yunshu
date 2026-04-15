import{j as a}from"./react-vendor-D2ezaHNm.js";import{Y as r}from"./yaml-crud-page-CxboUGeW.js";import{d as s,a as l,g as o,b as d}from"./rbac-miUcNleB.js";import{b as n}from"./antd-vendor-DhwHsd1P.js";import"./vendor-NnpWLRpG.js";import"./clusters-DEB4rssq.js";import"./index-CW0KuSxy.js";import"./shared-vendor-DEQMwG20.js";function f(){const i=[{title:"名称",dataIndex:"name",width:280},{title:"规则数",dataIndex:"rules",width:100,render:e=>a.jsx(n,{color:"blue",children:e})},{title:"创建时间",dataIndex:"creation_time",width:180,fixed:"right"}];return a.jsx(r,{title:"RBAC - ClusterRole",needNamespace:!1,columns:i,api:{list:async({clusterId:e,keyword:t})=>(await d(e,t)).list,detail:async({clusterId:e,name:t})=>await o({cluster_id:e,kind:"ClusterRole",name:t}),apply:async({clusterId:e,manifest:t})=>await l(e,t),remove:async({clusterId:e,name:t})=>await s({cluster_id:e,kind:"ClusterRole",name:t})},createTemplate:()=>`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: demo-clusterrole
rules:
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get", "list"]
`})}export{f as RbacClusterRolesPage};
