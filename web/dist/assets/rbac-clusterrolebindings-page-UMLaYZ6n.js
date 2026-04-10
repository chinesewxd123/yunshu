import{j as a}from"./react-vendor-CZTxzUXK.js";import{Y as s}from"./yaml-crud-page-Ncx6ELVy.js";import{d as n,a as l,g as d,l as o}from"./rbac-DjrB7kph.js";import{b as i,r as c,T as m}from"./antd-vendor-DHV5ALct.js";import"./vendor-DACzfRyF.js";import"./clusters-P6dsZFUi.js";import"./index-B1IehXz1.js";import"./shared-vendor-eQNmuKbk.js";function f(){const r=[{title:"名称",dataIndex:"name",width:280},{title:"RoleRef",dataIndex:"role_ref",width:240,render:e=>a.jsx(i,{children:e||"-"})},{title:"Subjects",dataIndex:"subjects",render:e=>e!=null&&e.length?a.jsxs(c,{wrap:!0,size:[6,6],children:[e.slice(0,8).map(t=>a.jsx(i,{children:t},t)),e.length>8?a.jsxs(m.Text,{type:"secondary",children:["+",e.length-8]}):null]}):a.jsx("span",{className:"inline-muted",children:"-"})},{title:"创建时间",dataIndex:"creation_time",width:180,fixed:"right"}];return a.jsx(s,{title:"RBAC - ClusterRoleBinding",needNamespace:!1,columns:r,api:{list:async({clusterId:e,keyword:t})=>(await o(e,t)).list,detail:async({clusterId:e,name:t})=>await d({cluster_id:e,kind:"ClusterRoleBinding",name:t}),apply:async({clusterId:e,manifest:t})=>await l(e,t),remove:async({clusterId:e,name:t})=>await n({cluster_id:e,kind:"ClusterRoleBinding",name:t})},createTemplate:()=>`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: demo-clusterrolebinding
subjects:
  - kind: User
    name: demo-user
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: demo-clusterrole
`})}export{f as RbacClusterRoleBindingsPage};
