import{r as n,j as a}from"./react-vendor-CZTxzUXK.js";import{Y as w}from"./yaml-crud-page-Ncx6ELVy.js";import{l as f}from"./clusters-P6dsZFUi.js";import{r as y,b as I,c as k,e as j,f as T}from"./ingresses-BynCQJCT.js";import{B as l,K as d,s as b,Q as v,T as r,ar as K,ag as N,W as P}from"./antd-vendor-DHV5ALct.js";import"./vendor-DACzfRyF.js";import"./index-B1IehXz1.js";import"./shared-vendor-eQNmuKbk.js";function E(){const[c,i]=n.useState(!1),[p,m]=n.useState("详情"),[x,g]=n.useState({}),h=(e,t)=>{m(e),g(t??{}),i(!0)},o=(e,t,s)=>a.jsx(P,{title:e,children:a.jsx(l,{type:"link",size:"small",icon:t,onClick:()=>h(e,s)})}),u=[{title:"命名空间",dataIndex:"namespace",width:120},{title:"名称",dataIndex:"name",width:180},{title:"访问规则",dataIndex:"rules_text",width:320,render:e=>a.jsx(r.Text,{style:{whiteSpace:"pre-wrap",fontSize:12},children:e||"-"})},{title:"标签",key:"labels",width:70,align:"center",render:(e,t)=>o("标签",a.jsx(K,{}),t.labels)},{title:"注解",key:"annotations",width:70,align:"center",render:(e,t)=>o("注解",a.jsx(N,{}),t.annotations)},{title:"入口控制器",dataIndex:"class_name",width:180,render:e=>e||"-"},{title:"LB地址",dataIndex:"load_balancer",width:180,render:e=>e||"-"},{title:"存在时长",dataIndex:"age",width:90,fixed:"right",render:e=>e||"-"},{title:"创建时间",dataIndex:"creation_time",width:180,fixed:"right"}];return a.jsxs(a.Fragment,{children:[a.jsx(w,{title:"Ingress-Nginx 管理",needNamespace:!0,onLoadNamespaces:async e=>((await f(e)).list??[]).map(s=>({label:s.name,value:s.name})),columns:u,api:{list:async({clusterId:e,namespace:t,keyword:s})=>await T(e,t??"default",s),detail:async({clusterId:e,namespace:t,name:s})=>await j(e,t??"default",s),apply:async({clusterId:e,manifest:t})=>await k(e,t),remove:async({clusterId:e,namespace:t,name:s})=>await I(e,t??"default",s)},renderToolbarExtraRight:({clusterId:e,reload:t})=>a.jsx(l,{disabled:!e,onClick:()=>{e&&d.confirm({title:"重启 Ingress-Nginx Controller Pods",content:"将删除 ingress-nginx controller Pods 以触发自动重建，用于刷新默认证书等运行态资源。确认继续吗？",okText:"确认重启",cancelText:"取消",onOk:async()=>{const s=await y(e);b.success(`已删除 ${s.deleted_count} 个 Pod`),await t()}})},children:"重启 Ingress-Nginx"}),createTemplate:({namespace:e})=>`apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: demo-ingress
  namespace: ${e??"default"}
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  ingressClassName: nginx
  rules:
    - host: demo.local
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: demo-service
                port:
                  number: 80
`}),a.jsx(d,{title:p,open:c,onCancel:()=>i(!1),footer:null,width:720,children:a.jsx(v,{rowKey:e=>e.key,pagination:!1,dataSource:Object.entries(x).map(([e,t])=>({key:e,value:t})),locale:{emptyText:"暂无数据"},columns:[{title:"Key",dataIndex:"key",width:260,render:e=>a.jsx(r.Text,{copyable:!0,children:e})},{title:"Value",dataIndex:"value",render:e=>a.jsx(r.Text,{copyable:!0,style:{whiteSpace:"pre-wrap"},children:e})}]})})]})}export{E as IngressesPage};
