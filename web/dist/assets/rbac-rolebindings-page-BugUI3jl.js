import{j as i}from"./react-vendor-D2ezaHNm.js";import{Y as r}from"./yaml-crud-page-CxboUGeW.js";import{l}from"./clusters-DEB4rssq.js";import{d,a as o,g as m,c}from"./rbac-miUcNleB.js";import{b as n,r as p,T as u}from"./antd-vendor-DhwHsd1P.js";import"./vendor-NnpWLRpG.js";import"./index-CW0KuSxy.js";import"./shared-vendor-DEQMwG20.js";function w(){const s=[{title:"名称",dataIndex:"name",width:260},{title:"RoleRef",dataIndex:"role_ref",width:240,render:e=>i.jsx(n,{children:e||"-"})},{title:"Subjects",dataIndex:"subjects",render:e=>e!=null&&e.length?i.jsxs(p,{wrap:!0,size:[6,6],children:[e.slice(0,8).map(a=>i.jsx(n,{children:a},a)),e.length>8?i.jsxs(u.Text,{type:"secondary",children:["+",e.length-8]}):null]}):i.jsx("span",{className:"inline-muted",children:"-"})},{title:"创建时间",dataIndex:"creation_time",width:180,fixed:"right"}];return i.jsx(r,{title:"RBAC - RoleBinding",needNamespace:!0,onLoadNamespaces:async e=>((await l(e)).list??[]).map(t=>({label:t.name,value:t.name})),columns:s,api:{list:async({clusterId:e,namespace:a,keyword:t})=>(await c(e,a??"default",t)).list,detail:async({clusterId:e,namespace:a,name:t})=>await m({cluster_id:e,kind:"RoleBinding",namespace:a??"default",name:t}),apply:async({clusterId:e,manifest:a})=>await o(e,a),remove:async({clusterId:e,namespace:a,name:t})=>await d({cluster_id:e,kind:"RoleBinding",namespace:a??"default",name:t})},createTemplate:({namespace:e})=>`apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: demo-rolebinding
  namespace: ${e||"default"}
subjects:
  - kind: User
    name: demo-user
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: demo-role
`})}export{w as RbacRoleBindingsPage};
