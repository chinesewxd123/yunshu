import{j as r}from"./react-vendor-D2ezaHNm.js";import{Y as d}from"./yaml-crud-page-CxboUGeW.js";import{k as s,c as o}from"./service-factory-BxxxI9TW.js";import{b as n}from"./antd-vendor-DhwHsd1P.js";import"./vendor-NnpWLRpG.js";import"./clusters-DEB4rssq.js";import"./index-CW0KuSxy.js";import"./shared-vendor-DEQMwG20.js";const a=o("/crds");function p(t,e){return a.list(s(t,{keyword:e}))}function c(t,e){return a.detail(s(t,{name:e}))}function l(t,e){return a.apply({cluster_id:t,manifest:e})}function m(t,e){return a.remove(s(t,{name:e}))}function I(){const t=[{title:"名称",dataIndex:"name",width:280},{title:"Group",dataIndex:"group",width:220},{title:"Kind",dataIndex:"kind",width:160},{title:"Plural",dataIndex:"plural",width:160},{title:"作用域",dataIndex:"scope",width:120},{title:"版本",dataIndex:"current_version",width:120},{title:"状态",dataIndex:"established",width:100,render:e=>e?r.jsx(n,{color:"green",children:"已建立"}):r.jsx(n,{color:"orange",children:"未就绪"})},{title:"创建时间",dataIndex:"creation_time",width:180,fixed:"right"}];return r.jsx(d,{title:"CRD 管理",columns:t,api:{list:async({clusterId:e,keyword:i})=>await p(e,i),detail:async({clusterId:e,name:i})=>await c(e,i),apply:async({clusterId:e,manifest:i})=>await l(e,i),remove:async({clusterId:e,name:i})=>await m(e,i)},createTemplate:()=>`apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: widgets.demo.example.com
spec:
  group: demo.example.com
  scope: Namespaced
  names:
    plural: widgets
    singular: widget
    kind: Widget
    shortNames:
      - wd
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                size:
                  type: string
`})}export{I as CrdsPage};
