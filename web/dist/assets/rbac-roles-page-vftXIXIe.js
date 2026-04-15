import{j as s}from"./react-vendor-D2ezaHNm.js";import{Y as r}from"./yaml-crud-page-CxboUGeW.js";import{l}from"./clusters-DEB4rssq.js";import{d as o,a as m,g as n,e as d}from"./rbac-miUcNleB.js";import{b as c}from"./antd-vendor-DhwHsd1P.js";import"./vendor-NnpWLRpG.js";import"./index-CW0KuSxy.js";import"./shared-vendor-DEQMwG20.js";function y(){const i=[{title:"名称",dataIndex:"name",width:260},{title:"规则数",dataIndex:"rules",width:100,render:a=>s.jsx(c,{color:"blue",children:a})},{title:"创建时间",dataIndex:"creation_time",width:180,fixed:"right"}];return s.jsx(r,{title:"RBAC - Role",needNamespace:!0,onLoadNamespaces:async a=>((await l(a)).list??[]).map(t=>({label:t.name,value:t.name})),columns:i,api:{list:async({clusterId:a,namespace:e,keyword:t})=>(await d(a,e??"default",t)).list,detail:async({clusterId:a,namespace:e,name:t})=>await n({cluster_id:a,kind:"Role",namespace:e??"default",name:t}),apply:async({clusterId:a,manifest:e})=>await m(a,e),remove:async({clusterId:a,namespace:e,name:t})=>await o({cluster_id:a,kind:"Role",namespace:e??"default",name:t})},createTemplate:({namespace:a})=>`apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: demo-role
  namespace: ${a||"default"}
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list"]
`})}export{y as RbacRolesPage};
