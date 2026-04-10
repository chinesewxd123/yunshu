import{j as n}from"./react-vendor-CZTxzUXK.js";import{Y as o}from"./yaml-crud-page-Ncx6ELVy.js";import{l as s}from"./clusters-P6dsZFUi.js";import{d as p,a as m,g as r,l}from"./configs-D2fHSfZx.js";import"./vendor-DACzfRyF.js";import"./antd-vendor-DHV5ALct.js";import"./shared-vendor-eQNmuKbk.js";import"./index-B1IehXz1.js";function y(){const i=[{title:"名称",dataIndex:"name"},{title:"键数量",dataIndex:"data_count",width:120},{title:"创建时间",dataIndex:"creation_time",width:180,fixed:"right"}];return n.jsx(o,{title:"ConfigMap 管理",needNamespace:!0,onLoadNamespaces:async a=>((await s(a)).list??[]).map(e=>({label:e.name,value:e.name})),columns:i,api:{list:async({clusterId:a,namespace:t,keyword:e})=>await l(a,t??"default",e),detail:async({clusterId:a,namespace:t,name:e})=>await r(a,t??"default",e),apply:async({clusterId:a,manifest:t})=>await m(a,t),remove:async({clusterId:a,namespace:t,name:e})=>await p(a,t??"default",e)},createTemplate:({namespace:a})=>`apiVersion: v1
kind: ConfigMap
metadata:
  name: demo-config
  namespace: ${a??"default"}
data:
  app.env: "prod"
  feature.flag: "true"
`})}export{y as ConfigmapsPage};
