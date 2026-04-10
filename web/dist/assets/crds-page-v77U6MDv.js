import{j as d}from"./react-vendor-CZTxzUXK.js";import{Y as n}from"./yaml-crud-page-Ncx6ELVy.js";import{g as r,h as i}from"./index-B1IehXz1.js";import{b as s}from"./antd-vendor-DHV5ALct.js";import"./vendor-DACzfRyF.js";import"./clusters-P6dsZFUi.js";import"./shared-vendor-eQNmuKbk.js";function o(t,e){return r(i.get("/crds",{params:{cluster_id:t,keyword:e}}))}function p(t,e){return r(i.get("/crds/detail",{params:{cluster_id:t,name:e}}))}function l(t,e){return r(i.post("/crds/apply",{cluster_id:t,manifest:e}))}function c(t,e){return r(i.delete("/crds",{params:{cluster_id:t,name:e}}))}function y(){const t=[{title:"名称",dataIndex:"name",width:280},{title:"Group",dataIndex:"group",width:220},{title:"Kind",dataIndex:"kind",width:160},{title:"Plural",dataIndex:"plural",width:160},{title:"作用域",dataIndex:"scope",width:120},{title:"版本",dataIndex:"current_version",width:120},{title:"状态",dataIndex:"established",width:100,render:e=>e?d.jsx(s,{color:"green",children:"已建立"}):d.jsx(s,{color:"orange",children:"未就绪"})},{title:"创建时间",dataIndex:"creation_time",width:180,fixed:"right"}];return d.jsx(n,{title:"CRD 管理",columns:t,api:{list:async({clusterId:e,keyword:a})=>await o(e,a),detail:async({clusterId:e,name:a})=>await p(e,a),apply:async({clusterId:e,manifest:a})=>await l(e,a),remove:async({clusterId:e,name:a})=>await c(e,a)},createTemplate:()=>`apiVersion: apiextensions.k8s.io/v1
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
`})}export{y as CrdsPage};
