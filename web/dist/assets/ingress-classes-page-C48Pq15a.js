import{j as t}from"./react-vendor-D2ezaHNm.js";import{u as r}from"./key-value-viewer-BErqQYVJ.js";import{Y as l}from"./yaml-crud-page-CxboUGeW.js";import{d as o,a as d,g,l as m}from"./ingresses-Dnm6IZfk.js";import{b as c,as as p,ag as x}from"./antd-vendor-DhwHsd1P.js";import"./vendor-NnpWLRpG.js";import"./clusters-DEB4rssq.js";import"./index-CW0KuSxy.js";import"./shared-vendor-DEQMwG20.js";import"./service-factory-BxxxI9TW.js";function b(){const{renderKVIcon:a,viewer:n}=r(),i=[{title:"名称",dataIndex:"name",width:180},{title:"Ingress数量",dataIndex:"ingress_count",width:120},{title:"控制器名称",dataIndex:"controller",width:260,render:e=>e||"-"},{title:"是否默认",dataIndex:"is_default",width:110,render:e=>t.jsx(c,{color:e?"green":"default",children:e?"是":"否"})},{title:"标签",key:"labels",width:70,align:"center",render:(e,s)=>a("标签",t.jsx(p,{}),s.labels)},{title:"注解",key:"annotations",width:70,align:"center",render:(e,s)=>a("注解",t.jsx(x,{}),s.annotations)},{title:"存在时长",dataIndex:"age",width:100,fixed:"right",render:e=>e||"-"},{title:"创建时间",dataIndex:"creation_time",width:180,fixed:"right"}];return t.jsxs(t.Fragment,{children:[t.jsx(l,{title:"IngressClass 入口类管理",columns:i,api:{list:async({clusterId:e,keyword:s})=>await m(e,s),detail:async({clusterId:e,name:s})=>await g(e,s),apply:async({clusterId:e,manifest:s})=>await d(e,s),remove:async({clusterId:e,name:s})=>await o(e,s)},createTemplate:()=>`apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: nginx
  annotations:
    ingressclass.kubernetes.io/is-default-class: "false"
spec:
  controller: k8s.io/ingress-nginx
`}),n]})}export{b as IngressClassesPage};
