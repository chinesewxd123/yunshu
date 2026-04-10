import{j as r}from"./react-vendor-CZTxzUXK.js";import{Y as d}from"./yaml-crud-page-Ncx6ELVy.js";import{l as n}from"./clusters-P6dsZFUi.js";import{b as l,c,e as o,f as m}from"./configs-D2fHSfZx.js";import{ae as p,T as i}from"./antd-vendor-DHV5ALct.js";import"./vendor-DACzfRyF.js";import"./index-B1IehXz1.js";import"./shared-vendor-eQNmuKbk.js";function S(){const s=[{title:"名称",dataIndex:"name"},{title:"类型",dataIndex:"type",width:180},{title:"键数量",dataIndex:"data_count",width:120},{title:"创建时间",dataIndex:"creation_time",width:180,fixed:"right"}];return r.jsx(d,{title:"Secret 管理",needNamespace:!0,onLoadNamespaces:async e=>((await n(e)).list??[]).map(t=>({label:t.name,value:t.name})),columns:s,api:{list:async({clusterId:e,namespace:a,keyword:t})=>await m(e,a??"default",t),detail:async({clusterId:e,namespace:a,name:t})=>await o(e,a??"default",t),apply:async({clusterId:e,manifest:a})=>await c(e,a),remove:async({clusterId:e,namespace:a,name:t})=>await l(e,a??"default",t)},createTemplate:({namespace:e})=>`apiVersion: v1
kind: Secret
metadata:
  name: demo-secret
  namespace: ${e??"default"}
type: Opaque
stringData:
  username: admin
  password: Admin@123
`,detailExtra:e=>r.jsxs("div",{children:[r.jsx(p,{type:"warning",showIcon:!0,message:"注意",description:"Secret 中的 data 可能包含敏感信息。下方 YAML 会包含 base64 内容；decoded_data 仅用于调试查看。"}),e.decoded_data?r.jsxs("div",{style:{marginTop:12},children:[r.jsx(i.Title,{level:5,style:{marginTop:0},children:"decoded_data（仅供查看）"}),r.jsx(i.Paragraph,{style:{whiteSpace:"pre-wrap"},children:Object.entries(e.decoded_data).map(([a,t])=>`${a}: ${t}`).join(`
`)})]}):null]})})}export{S as SecretsPage};
