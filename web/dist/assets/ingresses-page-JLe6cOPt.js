import{j as s}from"./react-vendor-D2ezaHNm.js";import{u as o}from"./key-value-viewer-BErqQYVJ.js";import{Y as d}from"./yaml-crud-page-CxboUGeW.js";import{l}from"./clusters-DEB4rssq.js";import{r as c,b as m,c as p,e as g,f as x}from"./ingresses-Dnm6IZfk.js";import{B as h,K as w,s as u,T as f,as as I,ag as y}from"./antd-vendor-DhwHsd1P.js";import"./vendor-NnpWLRpG.js";import"./index-CW0KuSxy.js";import"./shared-vendor-DEQMwG20.js";import"./service-factory-BxxxI9TW.js";function V(){const{renderKVIcon:n,viewer:r}=o(),i=[{title:"命名空间",dataIndex:"namespace",width:120},{title:"名称",dataIndex:"name",width:180},{title:"访问规则",dataIndex:"rules_text",width:320,render:e=>s.jsx(f.Text,{style:{whiteSpace:"pre-wrap",fontSize:12},children:e||"-"})},{title:"标签",key:"labels",width:70,align:"center",render:(e,t)=>n("标签",s.jsx(I,{}),t.labels)},{title:"注解",key:"annotations",width:70,align:"center",render:(e,t)=>n("注解",s.jsx(y,{}),t.annotations)},{title:"入口控制器",dataIndex:"class_name",width:180,render:e=>e||"-"},{title:"LB地址",dataIndex:"load_balancer",width:180,render:e=>e||"-"},{title:"存在时长",dataIndex:"age",width:90,fixed:"right",render:e=>e||"-"},{title:"创建时间",dataIndex:"creation_time",width:180,fixed:"right"}];return s.jsxs(s.Fragment,{children:[s.jsx(d,{title:"Ingress-Nginx 管理",needNamespace:!0,onLoadNamespaces:async e=>((await l(e)).list??[]).map(a=>({label:a.name,value:a.name})),columns:i,api:{list:async({clusterId:e,namespace:t,keyword:a})=>await x(e,t??"default",a),detail:async({clusterId:e,namespace:t,name:a})=>await g(e,t??"default",a),apply:async({clusterId:e,manifest:t})=>await p(e,t),remove:async({clusterId:e,namespace:t,name:a})=>await m(e,t??"default",a)},renderToolbarExtraRight:({clusterId:e,reload:t})=>s.jsx(h,{disabled:!e,onClick:()=>{e&&w.confirm({title:"重启 Ingress-Nginx Controller Pods",content:"将删除 ingress-nginx controller Pods 以触发自动重建，用于刷新默认证书等运行态资源。确认继续吗？",okText:"确认重启",cancelText:"取消",onOk:async()=>{const a=await c(e);u.success(`已删除 ${a.deleted_count} 个 Pod`),await t()}})},children:"重启 Ingress-Nginx"}),createTemplate:({namespace:e})=>`apiVersion: networking.k8s.io/v1
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
`}),r]})}export{V as IngressesPage};
