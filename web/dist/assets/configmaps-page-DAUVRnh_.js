import{j as n}from"./react-vendor-D2ezaHNm.js";import{Y as o}from"./yaml-crud-page-CxboUGeW.js";import{l as s}from"./clusters-DEB4rssq.js";import{d as p,a as m,g as r,l}from"./configs-D-CfWmau.js";import"./vendor-NnpWLRpG.js";import"./antd-vendor-DhwHsd1P.js";import"./shared-vendor-DEQMwG20.js";import"./index-CW0KuSxy.js";function y(){const i=[{title:"名称",dataIndex:"name"},{title:"键数量",dataIndex:"data_count",width:120},{title:"创建时间",dataIndex:"creation_time",width:180,fixed:"right"}];return n.jsx(o,{title:"ConfigMap 管理",needNamespace:!0,onLoadNamespaces:async a=>((await s(a)).list??[]).map(e=>({label:e.name,value:e.name})),columns:i,api:{list:async({clusterId:a,namespace:t,keyword:e})=>await l(a,t??"default",e),detail:async({clusterId:a,namespace:t,name:e})=>await r(a,t??"default",e),apply:async({clusterId:a,manifest:t})=>await m(a,t),remove:async({clusterId:a,namespace:t,name:e})=>await p(a,t??"default",e)},createTemplate:({namespace:a})=>`apiVersion: v1
kind: ConfigMap
metadata:
  name: demo-config
  namespace: ${a??"default"}
data:
  app.env: "prod"
  feature.flag: "true"
`})}export{y as ConfigmapsPage};
