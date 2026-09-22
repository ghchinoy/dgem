(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const i of document.querySelectorAll('link[rel="modulepreload"]'))s(i);new MutationObserver(i=>{for(const o of i)if(o.type==="childList")for(const n of o.addedNodes)n.tagName==="LINK"&&n.rel==="modulepreload"&&s(n)}).observe(document,{childList:!0,subtree:!0});function e(i){const o={};return i.integrity&&(o.integrity=i.integrity),i.referrerPolicy&&(o.referrerPolicy=i.referrerPolicy),i.crossOrigin==="use-credentials"?o.credentials="include":i.crossOrigin==="anonymous"?o.credentials="omit":o.credentials="same-origin",o}function s(i){if(i.ep)return;i.ep=!0;const o=e(i);fetch(i.href,o)}})();/**
 * @license
 * Copyright 2019 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */const I=globalThis,L=I.ShadowRoot&&(I.ShadyCSS===void 0||I.ShadyCSS.nativeShadow)&&"adoptedStyleSheets"in Document.prototype&&"replace"in CSSStyleSheet.prototype,j=Symbol(),W=new WeakMap;let it=class{constructor(t,e,s){if(this._$cssResult$=!0,s!==j)throw Error("CSSResult is not constructable. Use `unsafeCSS` or `css` instead.");this.cssText=t,this.t=e}get styleSheet(){let t=this.o;const e=this.t;if(L&&t===void 0){const s=e!==void 0&&e.length===1;s&&(t=W.get(e)),t===void 0&&((this.o=t=new CSSStyleSheet).replaceSync(this.cssText),s&&W.set(e,t))}return t}toString(){return this.cssText}};const ct=r=>new it(typeof r=="string"?r:r+"",void 0,j),ht=(r,...t)=>{const e=r.length===1?r[0]:t.reduce((s,i,o)=>s+(n=>{if(n._$cssResult$===!0)return n.cssText;if(typeof n=="number")return n;throw Error("Value passed to 'css' function must be a 'css' function result: "+n+". Use 'unsafeCSS' to pass non-literal values, but take care to ensure page security.")})(i)+r[o+1],r[0]);return new it(e,r,j)},dt=(r,t)=>{if(L)r.adoptedStyleSheets=t.map(e=>e instanceof CSSStyleSheet?e:e.styleSheet);else for(const e of t){const s=document.createElement("style"),i=I.litNonce;i!==void 0&&s.setAttribute("nonce",i),s.textContent=e.cssText,r.appendChild(s)}},F=L?r=>r:r=>r instanceof CSSStyleSheet?(t=>{let e="";for(const s of t.cssRules)e+=s.cssText;return ct(e)})(r):r;/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */const{is:pt,defineProperty:ut,getOwnPropertyDescriptor:ft,getOwnPropertyNames:gt,getOwnPropertySymbols:$t,getPrototypeOf:mt}=Object,k=globalThis,G=k.trustedTypes,yt=G?G.emptyScript:"",bt=k.reactiveElementPolyfillSupport,C=(r,t)=>r,D={toAttribute(r,t){switch(t){case Boolean:r=r?yt:null;break;case Object:case Array:r=r==null?r:JSON.stringify(r)}return r},fromAttribute(r,t){let e=r;switch(t){case Boolean:e=r!==null;break;case Number:e=r===null?null:Number(r);break;case Object:case Array:try{e=JSON.parse(r)}catch{e=null}}return e}},B=(r,t)=>!pt(r,t),Z={attribute:!0,type:String,converter:D,reflect:!1,useDefault:!1,hasChanged:B};Symbol.metadata??=Symbol("metadata"),k.litPropertyMetadata??=new WeakMap;let A=class extends HTMLElement{static addInitializer(t){this._$Ei(),(this.l??=[]).push(t)}static get observedAttributes(){return this.finalize(),this._$Eh&&[...this._$Eh.keys()]}static createProperty(t,e=Z){if(e.state&&(e.attribute=!1),this._$Ei(),this.prototype.hasOwnProperty(t)&&((e=Object.create(e)).wrapped=!0),this.elementProperties.set(t,e),!e.noAccessor){const s=Symbol(),i=this.getPropertyDescriptor(t,s,e);i!==void 0&&ut(this.prototype,t,i)}}static getPropertyDescriptor(t,e,s){const{get:i,set:o}=ft(this.prototype,t)??{get(){return this[e]},set(n){this[e]=n}};return{get:i,set(n){const l=i?.call(this);o?.call(this,n),this.requestUpdate(t,l,s)},configurable:!0,enumerable:!0}}static getPropertyOptions(t){return this.elementProperties.get(t)??Z}static _$Ei(){if(this.hasOwnProperty(C("elementProperties")))return;const t=mt(this);t.finalize(),t.l!==void 0&&(this.l=[...t.l]),this.elementProperties=new Map(t.elementProperties)}static finalize(){if(this.hasOwnProperty(C("finalized")))return;if(this.finalized=!0,this._$Ei(),this.hasOwnProperty(C("properties"))){const e=this.properties,s=[...gt(e),...$t(e)];for(const i of s)this.createProperty(i,e[i])}const t=this[Symbol.metadata];if(t!==null){const e=litPropertyMetadata.get(t);if(e!==void 0)for(const[s,i]of e)this.elementProperties.set(s,i)}this._$Eh=new Map;for(const[e,s]of this.elementProperties){const i=this._$Eu(e,s);i!==void 0&&this._$Eh.set(i,e)}this.elementStyles=this.finalizeStyles(this.styles)}static finalizeStyles(t){const e=[];if(Array.isArray(t)){const s=new Set(t.flat(1/0).reverse());for(const i of s)e.unshift(F(i))}else t!==void 0&&e.push(F(t));return e}static _$Eu(t,e){const s=e.attribute;return s===!1?void 0:typeof s=="string"?s:typeof t=="string"?t.toLowerCase():void 0}constructor(){super(),this._$Ep=void 0,this.isUpdatePending=!1,this.hasUpdated=!1,this._$Em=null,this._$Ev()}_$Ev(){this._$ES=new Promise(t=>this.enableUpdating=t),this._$AL=new Map,this._$E_(),this.requestUpdate(),this.constructor.l?.forEach(t=>t(this))}addController(t){(this._$EO??=new Set).add(t),this.renderRoot!==void 0&&this.isConnected&&t.hostConnected?.()}removeController(t){this._$EO?.delete(t)}_$E_(){const t=new Map,e=this.constructor.elementProperties;for(const s of e.keys())this.hasOwnProperty(s)&&(t.set(s,this[s]),delete this[s]);t.size>0&&(this._$Ep=t)}createRenderRoot(){const t=this.shadowRoot??this.attachShadow(this.constructor.shadowRootOptions);return dt(t,this.constructor.elementStyles),t}connectedCallback(){this.renderRoot??=this.createRenderRoot(),this.enableUpdating(!0),this._$EO?.forEach(t=>t.hostConnected?.())}enableUpdating(t){}disconnectedCallback(){this._$EO?.forEach(t=>t.hostDisconnected?.())}attributeChangedCallback(t,e,s){this._$AK(t,s)}_$ET(t,e){const s=this.constructor.elementProperties.get(t),i=this.constructor._$Eu(t,s);if(i!==void 0&&s.reflect===!0){const o=(s.converter?.toAttribute!==void 0?s.converter:D).toAttribute(e,s.type);this._$Em=t,o==null?this.removeAttribute(i):this.setAttribute(i,o),this._$Em=null}}_$AK(t,e){const s=this.constructor,i=s._$Eh.get(t);if(i!==void 0&&this._$Em!==i){const o=s.getPropertyOptions(i),n=typeof o.converter=="function"?{fromAttribute:o.converter}:o.converter?.fromAttribute!==void 0?o.converter:D;this._$Em=i;const l=n.fromAttribute(e,o.type);this[i]=l??this._$Ej?.get(i)??l,this._$Em=null}}requestUpdate(t,e,s,i=!1,o){if(t!==void 0){const n=this.constructor;if(i===!1&&(o=this[t]),s??=n.getPropertyOptions(t),!((s.hasChanged??B)(o,e)||s.useDefault&&s.reflect&&o===this._$Ej?.get(t)&&!this.hasAttribute(n._$Eu(t,s))))return;this.C(t,e,s)}this.isUpdatePending===!1&&(this._$ES=this._$EP())}C(t,e,{useDefault:s,reflect:i,wrapped:o},n){s&&!(this._$Ej??=new Map).has(t)&&(this._$Ej.set(t,n??e??this[t]),o!==!0||n!==void 0)||(this._$AL.has(t)||(this.hasUpdated||s||(e=void 0),this._$AL.set(t,e)),i===!0&&this._$Em!==t&&(this._$Eq??=new Set).add(t))}async _$EP(){this.isUpdatePending=!0;try{await this._$ES}catch(e){Promise.reject(e)}const t=this.scheduleUpdate();return t!=null&&await t,!this.isUpdatePending}scheduleUpdate(){return this.performUpdate()}performUpdate(){if(!this.isUpdatePending)return;if(!this.hasUpdated){if(this.renderRoot??=this.createRenderRoot(),this._$Ep){for(const[i,o]of this._$Ep)this[i]=o;this._$Ep=void 0}const s=this.constructor.elementProperties;if(s.size>0)for(const[i,o]of s){const{wrapped:n}=o,l=this[i];n!==!0||this._$AL.has(i)||l===void 0||this.C(i,void 0,o,l)}}let t=!1;const e=this._$AL;try{t=this.shouldUpdate(e),t?(this.willUpdate(e),this._$EO?.forEach(s=>s.hostUpdate?.()),this.update(e)):this._$EM()}catch(s){throw t=!1,this._$EM(),s}t&&this._$AE(e)}willUpdate(t){}_$AE(t){this._$EO?.forEach(e=>e.hostUpdated?.()),this.hasUpdated||(this.hasUpdated=!0,this.firstUpdated(t)),this.updated(t)}_$EM(){this._$AL=new Map,this.isUpdatePending=!1}get updateComplete(){return this.getUpdateComplete()}getUpdateComplete(){return this._$ES}shouldUpdate(t){return!0}update(t){this._$Eq&&=this._$Eq.forEach(e=>this._$ET(e,this[e])),this._$EM()}updated(t){}firstUpdated(t){}};A.elementStyles=[],A.shadowRootOptions={mode:"open"},A[C("elementProperties")]=new Map,A[C("finalized")]=new Map,bt?.({ReactiveElement:A}),(k.reactiveElementVersions??=[]).push("2.1.2");/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */const q=globalThis,K=r=>r,N=q.trustedTypes,X=N?N.createPolicy("lit-html",{createHTML:r=>r}):void 0,rt="$lit$",b=`lit$${Math.random().toFixed(9).slice(2)}$`,ot="?"+b,vt=`<${ot}>`,x=document,O=()=>x.createComment(""),U=r=>r===null||typeof r!="object"&&typeof r!="function",V=Array.isArray,_t=r=>V(r)||typeof r?.[Symbol.iterator]=="function",z=`[ 	
\f\r]`,E=/<(?:(!--|\/[^a-zA-Z])|(\/?[a-zA-Z][^>\s]*)|(\/?$))/g,Q=/-->/g,Y=/>/g,v=RegExp(`>|${z}(?:([^\\s"'>=/]+)(${z}*=${z}*(?:[^ 	
\f\r"'\`<>=]|("|')|))|$)`,"g"),tt=/'/g,et=/"/g,nt=/^(?:script|style|textarea|title)$/i,at=r=>(t,...e)=>({_$litType$:r,strings:t,values:e}),m=at(1),xt=at(2),w=Symbol.for("lit-noChange"),p=Symbol.for("lit-nothing"),st=new WeakMap,_=x.createTreeWalker(x,129);function lt(r,t){if(!V(r)||!r.hasOwnProperty("raw"))throw Error("invalid template strings array");return X!==void 0?X.createHTML(t):t}const At=(r,t)=>{const e=r.length-1,s=[];let i,o=t===2?"<svg>":t===3?"<math>":"",n=E;for(let l=0;l<e;l++){const a=r[l];let h,d,c=-1,$=0;for(;$<a.length&&(n.lastIndex=$,d=n.exec(a),d!==null);)$=n.lastIndex,n===E?d[1]==="!--"?n=Q:d[1]!==void 0?n=Y:d[2]!==void 0?(nt.test(d[2])&&(i=RegExp("</"+d[2],"g")),n=v):d[3]!==void 0&&(n=v):n===v?d[0]===">"?(n=i??E,c=-1):d[1]===void 0?c=-2:(c=n.lastIndex-d[2].length,h=d[1],n=d[3]===void 0?v:d[3]==='"'?et:tt):n===et||n===tt?n=v:n===Q||n===Y?n=E:(n=v,i=void 0);const y=n===v&&r[l+1].startsWith("/>")?" ":"";o+=n===E?a+vt:c>=0?(s.push(h),a.slice(0,c)+rt+a.slice(c)+b+y):a+b+(c===-2?l:y)}return[lt(r,o+(r[e]||"<?>")+(t===2?"</svg>":t===3?"</math>":"")),s]};class M{constructor({strings:t,_$litType$:e},s){let i;this.parts=[];let o=0,n=0;const l=t.length-1,a=this.parts,[h,d]=At(t,e);if(this.el=M.createElement(h,s),_.currentNode=this.el.content,e===2||e===3){const c=this.el.content.firstChild;c.replaceWith(...c.childNodes)}for(;(i=_.nextNode())!==null&&a.length<l;){if(i.nodeType===1){if(i.hasAttributes())for(const c of i.getAttributeNames())if(c.endsWith(rt)){const $=d[n++],y=i.getAttribute(c).split(b),T=/([.?@])?(.*)/.exec($);a.push({type:1,index:o,name:T[2],strings:y,ctor:T[1]==="."?St:T[1]==="?"?Et:T[1]==="@"?Ct:H}),i.removeAttribute(c)}else c.startsWith(b)&&(a.push({type:6,index:o}),i.removeAttribute(c));if(nt.test(i.tagName)){const c=i.textContent.split(b),$=c.length-1;if($>0){i.textContent=N?N.emptyScript:"";for(let y=0;y<$;y++)i.append(c[y],O()),_.nextNode(),a.push({type:2,index:++o});i.append(c[$],O())}}}else if(i.nodeType===8)if(i.data===ot)a.push({type:2,index:o});else{let c=-1;for(;(c=i.data.indexOf(b,c+1))!==-1;)a.push({type:7,index:o}),c+=b.length-1}o++}}static createElement(t,e){const s=x.createElement("template");return s.innerHTML=t,s}}function S(r,t,e=r,s){if(t===w)return t;let i=s!==void 0?e._$Co?.[s]:e._$Cl;const o=U(t)?void 0:t._$litDirective$;return i?.constructor!==o&&(i?._$AO?.(!1),o===void 0?i=void 0:(i=new o(r),i._$AT(r,e,s)),s!==void 0?(e._$Co??=[])[s]=i:e._$Cl=i),i!==void 0&&(t=S(r,i._$AS(r,t.values),i,s)),t}class wt{constructor(t,e){this._$AV=[],this._$AN=void 0,this._$AD=t,this._$AM=e}get parentNode(){return this._$AM.parentNode}get _$AU(){return this._$AM._$AU}u(t){const{el:{content:e},parts:s}=this._$AD,i=(t?.creationScope??x).importNode(e,!0);_.currentNode=i;let o=_.nextNode(),n=0,l=0,a=s[0];for(;a!==void 0;){if(n===a.index){let h;a.type===2?h=new R(o,o.nextSibling,this,t):a.type===1?h=new a.ctor(o,a.name,a.strings,this,t):a.type===6&&(h=new Pt(o,this,t)),this._$AV.push(h),a=s[++l]}n!==a?.index&&(o=_.nextNode(),n++)}return _.currentNode=x,i}p(t){let e=0;for(const s of this._$AV)s!==void 0&&(s.strings!==void 0?(s._$AI(t,s,e),e+=s.strings.length-2):s._$AI(t[e])),e++}}class R{get _$AU(){return this._$AM?._$AU??this._$Cv}constructor(t,e,s,i){this.type=2,this._$AH=p,this._$AN=void 0,this._$AA=t,this._$AB=e,this._$AM=s,this.options=i,this._$Cv=i?.isConnected??!0}get parentNode(){let t=this._$AA.parentNode;const e=this._$AM;return e!==void 0&&t?.nodeType===11&&(t=e.parentNode),t}get startNode(){return this._$AA}get endNode(){return this._$AB}_$AI(t,e=this){t=S(this,t,e),U(t)?t===p||t==null||t===""?(this._$AH!==p&&this._$AR(),this._$AH=p):t!==this._$AH&&t!==w&&this._(t):t._$litType$!==void 0?this.$(t):t.nodeType!==void 0?this.T(t):_t(t)?this.k(t):this._(t)}O(t){return this._$AA.parentNode.insertBefore(t,this._$AB)}T(t){this._$AH!==t&&(this._$AR(),this._$AH=this.O(t))}_(t){this._$AH!==p&&U(this._$AH)?this._$AA.nextSibling.data=t:this.T(x.createTextNode(t)),this._$AH=t}$(t){const{values:e,_$litType$:s}=t,i=typeof s=="number"?this._$AC(t):(s.el===void 0&&(s.el=M.createElement(lt(s.h,s.h[0]),this.options)),s);if(this._$AH?._$AD===i)this._$AH.p(e);else{const o=new wt(i,this),n=o.u(this.options);o.p(e),this.T(n),this._$AH=o}}_$AC(t){let e=st.get(t.strings);return e===void 0&&st.set(t.strings,e=new M(t)),e}k(t){V(this._$AH)||(this._$AH=[],this._$AR());const e=this._$AH;let s,i=0;for(const o of t)i===e.length?e.push(s=new R(this.O(O()),this.O(O()),this,this.options)):s=e[i],s._$AI(o),i++;i<e.length&&(this._$AR(s&&s._$AB.nextSibling,i),e.length=i)}_$AR(t=this._$AA.nextSibling,e){for(this._$AP?.(!1,!0,e);t!==this._$AB;){const s=K(t).nextSibling;K(t).remove(),t=s}}setConnected(t){this._$AM===void 0&&(this._$Cv=t,this._$AP?.(t))}}class H{get tagName(){return this.element.tagName}get _$AU(){return this._$AM._$AU}constructor(t,e,s,i,o){this.type=1,this._$AH=p,this._$AN=void 0,this.element=t,this.name=e,this._$AM=i,this.options=o,s.length>2||s[0]!==""||s[1]!==""?(this._$AH=Array(s.length-1).fill(new String),this.strings=s):this._$AH=p}_$AI(t,e=this,s,i){const o=this.strings;let n=!1;if(o===void 0)t=S(this,t,e,0),n=!U(t)||t!==this._$AH&&t!==w,n&&(this._$AH=t);else{const l=t;let a,h;for(t=o[0],a=0;a<o.length-1;a++)h=S(this,l[s+a],e,a),h===w&&(h=this._$AH[a]),n||=!U(h)||h!==this._$AH[a],h===p?t=p:t!==p&&(t+=(h??"")+o[a+1]),this._$AH[a]=h}n&&!i&&this.j(t)}j(t){t===p?this.element.removeAttribute(this.name):this.element.setAttribute(this.name,t??"")}}class St extends H{constructor(){super(...arguments),this.type=3}j(t){this.element[this.name]=t===p?void 0:t}}class Et extends H{constructor(){super(...arguments),this.type=4}j(t){this.element.toggleAttribute(this.name,!!t&&t!==p)}}class Ct extends H{constructor(t,e,s,i,o){super(t,e,s,i,o),this.type=5}_$AI(t,e=this){if((t=S(this,t,e,0)??p)===w)return;const s=this._$AH,i=t===p&&s!==p||t.capture!==s.capture||t.once!==s.once||t.passive!==s.passive,o=t!==p&&(s===p||i);i&&this.element.removeEventListener(this.name,this,s),o&&this.element.addEventListener(this.name,this,t),this._$AH=t}handleEvent(t){typeof this._$AH=="function"?this._$AH.call(this.options?.host??this.element,t):this._$AH.handleEvent(t)}}class Pt{constructor(t,e,s){this.element=t,this.type=6,this._$AN=void 0,this._$AM=e,this.options=s}get _$AU(){return this._$AM._$AU}_$AI(t){S(this,t)}}const Ot=q.litHtmlPolyfillSupport;Ot?.(M,R),(q.litHtmlVersions??=[]).push("3.3.3");const Ut=(r,t,e)=>{const s=e?.renderBefore??t;let i=s._$litPart$;if(i===void 0){const o=e?.renderBefore??null;s._$litPart$=i=new R(t.insertBefore(O(),o),o,void 0,e??{})}return i._$AI(r),i};/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */const J=globalThis;class P extends A{constructor(){super(...arguments),this.renderOptions={host:this},this._$Do=void 0}createRenderRoot(){const t=super.createRenderRoot();return this.renderOptions.renderBefore??=t.firstChild,t}update(t){const e=this.render();this.hasUpdated||(this.renderOptions.isConnected=this.isConnected),super.update(t),this._$Do=Ut(e,this.renderRoot,this.renderOptions)}connectedCallback(){super.connectedCallback(),this._$Do?.setConnected(!0)}disconnectedCallback(){super.disconnectedCallback(),this._$Do?.setConnected(!1)}render(){return w}}P._$litElement$=!0,P.finalized=!0,J.litElementHydrateSupport?.({LitElement:P});const Mt=J.litElementPolyfillSupport;Mt?.({LitElement:P});(J.litElementVersions??=[]).push("4.2.2");/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */const Rt=r=>(t,e)=>{e!==void 0?e.addInitializer(()=>{customElements.define(r,t)}):customElements.define(r,t)};/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */const Tt={attribute:!0,type:String,converter:D,reflect:!1,hasChanged:B},It=(r=Tt,t,e)=>{const{kind:s,metadata:i}=e;let o=globalThis.litPropertyMetadata.get(i);if(o===void 0&&globalThis.litPropertyMetadata.set(i,o=new Map),s==="setter"&&((r=Object.create(r)).wrapped=!0),o.set(e.name,r),s==="accessor"){const{name:n}=e;return{set(l){const a=t.get.call(this);t.set.call(this,l),this.requestUpdate(n,a,r,!0,l)},init(l){return l!==void 0&&this.C(n,void 0,r,l),l}}}if(s==="setter"){const{name:n}=e;return function(l){const a=this[n];t.call(this,l),this.requestUpdate(n,a,r,!0,l)}}throw Error("Unsupported decorator location: "+s)};function Dt(r){return(t,e)=>typeof e=="object"?It(r,t,e):((s,i,o)=>{const n=i.hasOwnProperty(o);return i.constructor.createProperty(o,s),n?Object.getOwnPropertyDescriptor(i,o):void 0})(r,t,e)}/**
 * @license
 * Copyright 2017 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */function g(r){return Dt({...r,state:!0,attribute:!1})}var Nt=Object.defineProperty,kt=Object.getOwnPropertyDescriptor,f=(r,t,e,s)=>{for(var i=s>1?void 0:s?kt(t,e):t,o=r.length-1,n;o>=0;o--)(n=r[o])&&(i=(s?n(t,e,i):n(i))||i);return s&&i&&Nt(t,e,i),i};let u=class extends P{constructor(){super(...arguments),this.catalog=[],this.selectedIdx=0,this.vars={},this.rawTemplate="",this.uploadedDataURI="",this.status=null,this.loading=!1,this.elapsedSec=0,this.result=null,this.errorMsg=""}connectedCallback(){super.connectedCallback(),this.fetchStatus(),this.fetchCatalog()}async fetchStatus(){try{const r=await fetch("/api/status",{credentials:"same-origin"});r.ok&&(this.status=await r.json())}catch{}}async fetchCatalog(){try{const t=await(await fetch("/api/templates",{credentials:"same-origin"})).json();this.catalog=t.templates||[];const e=this.catalog.findIndex(s=>s.id==="support_triage");this.selectTemplate(e>=0?e:0)}catch(r){this.errorMsg=`Failed to load policy catalog: ${r.message}`}}selectTemplate(r){this.selectedIdx=r;const t=this.catalog[r];t&&(this.rawTemplate=t.raw_template,this.vars={...t.sample_vars||{}},this.result=null,this.errorMsg="")}onImageChange(r){const e=r.target.files?.[0];if(!e)return;const s=new FileReader;s.onload=()=>{this.uploadedDataURI=String(s.result||"")},s.readAsDataURL(e)}async executeDecision(){const r=this.catalog[this.selectedIdx];if(!r)return;this.loading=!0,this.errorMsg="",this.elapsedSec=0;const t=Date.now();this.timerId=window.setInterval(()=>{this.elapsedSec=Math.floor((Date.now()-t)/1e3)},1e3);try{const e=await fetch("/api/decide",{method:"POST",credentials:"same-origin",headers:{"Content-Type":"application/json"},body:JSON.stringify({template:r.id,custom_template:this.rawTemplate,variables:this.vars,image:this.uploadedDataURI||void 0})}),s=await e.json();e.ok?this.result=s:this.errorMsg=s.error||JSON.stringify(s,null,2)}catch(e){this.errorMsg=e.message}finally{this.timerId&&clearInterval(this.timerId),this.loading=!1,this.fetchStatus()}}renderBboxOverlay(){if(!this.result)return null;const r=this.result.answers||{},t=n=>{if(!n)return null;const l=n.label||n.value||n.choice;if(l==null)return null;const a=String(l).match(/(\d+)/);return a?parseInt(a[1],10):null},e=t(r.ymin),s=t(r.xmin),i=t(r.ymax),o=t(r.xmax);return e===null||s===null||i===null||o===null?null:xt`
      <rect
        x="${s}"
        y="${e}"
        width="${Math.max(1,o-s)}"
        height="${Math.max(1,i-e)}"
        fill="rgba(56, 189, 248, 0.18)"
        stroke="#38bdf8"
        stroke-width="8"
      />
    `}render(){const r=this.catalog[this.selectedIdx],t=window.location.origin,e=JSON.stringify({variables:this.vars},null,2);return m`
      <header>
        <div class="brand">
          <h1>DiffusionGemma Decision Studio</h1>
          <span class="badge">Lit WebComponents</span>
          <span class="badge">O(1) Discrete Diffusion Readout</span>
        </div>
        <div class="status-pill">
          <span class="dot ${this.status?.reachable?"ok":"warn"}"></span>
          <span>
            ${this.status?.reachable?`GPU Container Reachable (${this.status.upstream_url})`:"Idle / Scaled-to-Zero (Auto-wakes on first request)"}
          </span>
        </div>
      </header>

      <main>
        <div class="card">
          <h2>1. Select Policy Template (.json.tmpl)</h2>
          <label for="tmplSelect">Policy Catalog (${this.catalog.length} templates)</label>
          <select
            id="tmplSelect"
            @change=${s=>this.selectTemplate(Number(s.target.value))}
          >
            ${this.catalog.map((s,i)=>m`
                <option value=${i} ?selected=${i===this.selectedIdx}>
                  [${s.category}] ${s.id}
                </option>
              `)}
          </select>

          ${(r?.variables||[]).map(s=>m`
              <label>Variable: {{ .${s} }}</label>
              <textarea
                rows="2"
                .value=${this.vars[s]||""}
                @input=${i=>{this.vars={...this.vars,[s]:i.target.value}}}
              ></textarea>
            `)}

          ${r?.multimodal?m`
                <label>Attach Image (PNG/JPEG for Multimodal SigLIP BBox Readout)</label>
                <input type="file" accept="image/*" @change=${this.onImageChange} />
              `:null}

          <details style="margin-top:14px;">
            <summary style="cursor:pointer; font-size:12px; color:var(--muted);">
              View / Edit Raw .json.tmpl Policy Source
            </summary>
            <textarea
              rows="10"
              style="margin-top:8px;"
              .value=${this.rawTemplate}
              @input=${s=>{this.rawTemplate=s.target.value}}
            ></textarea>
          </details>

          <button class="primary" ?disabled=${this.loading} @click=${this.executeDecision}>
            ${this.loading?`⏳ Evaluating (${this.elapsedSec}s elapsed)...`:"⚡ Execute Joint Decision Readout"}
          </button>
        </div>

        <div class="card">
          <h2>2. Joint Slot Decisions & Epistemic Shannon Entropy (H)</h2>
          ${this.errorMsg?m`<pre style="color:var(--fail);">${this.errorMsg}</pre>`:this.result?this.renderDecisionResults(this.result):m`
                  <p style="color:var(--muted); font-size:13px;">
                    Select a policy on the left and click
                    <b>Execute Joint Decision Readout</b>. If the Cloud Run RTX Pro 6000
                    GPU is scaled to zero, this gateway automatically wakes it and holds
                    the request until vLLM completes warmup.
                  </p>
                `}

          ${this.uploadedDataURI?m`
                <div class="canvas-wrap">
                  <img src=${this.uploadedDataURI} alt="Uploaded preview" />
                  <svg viewBox="0 0 1000 1000" preserveAspectRatio="none">
                    ${this.renderBboxOverlay()}
                  </svg>
                </div>
              `:null}

          <h2 style="margin-top:22px;">3. Zero-CLI cURL & dgem Snippet</h2>
          <pre># Option A: Direct REST API call (No dgem CLI needed)
curl -s "${t}/api/decide/${r?.id||"support_triage"}" \\
  -H "Authorization: Bearer $(gcloud auth print-identity-token)" \\
  -H "Content-Type: application/json" \\
  -d '${e}' | jq .

# Option B: Using dgem CLI against this gateway
./bin/dgem decide -u "${t}/v1" --gcp-auth -t templates/${r?.id||"support_triage"}.json.tmpl -s</pre>
        </div>
      </main>
    `}renderDecisionResults(r){const t=r.answers||{},e=r.diagnostics?.questions||{},s=Object.keys(t).sort();return m`
      <div
        style="display:flex; gap:14px; font-size:12px; color:var(--muted); margin-bottom:10px; flex-wrap:wrap;"
      >
        <span>⏱️ Latency: <b style="color:var(--text)">${r.wall_time_ms} ms</b></span>
        <span>
          📊 Max Entropy (H):
          <b style="color:var(--text)">${(r.max_entropy||0).toFixed(4)} nats</b>
        </span>
        <span>
          🔄 Warmup Attempts: <b style="color:var(--text)">${r.warmup_attempts}</b>
        </span>
      </div>

      <table class="slot-table">
        <thead>
          <tr>
            <th>Slot</th>
            <th>Decision Value</th>
            <th>Confidence (P)</th>
            <th>Shannon Entropy (H)</th>
          </tr>
        </thead>
        <tbody>
          ${s.map(i=>{const o=t[i],n=e[i],l=o.label!==void 0&&o.label!==""?o.label:o.value!==void 0?o.value:o.choice,a=`${((o.confidence||0)*100).toFixed(1)}%`,h=o.entropy!==void 0&&o.entropy>0?o.entropy:n?.entropy!==void 0?n.entropy:0,d=h>=.35?"entropy-high":h>=.15?"entropy-mid":"entropy-low";return m`
              <tr>
                <td><b>${i}</b></td>
                <td style="color:var(--accent); font-weight:600;">
                  ${JSON.stringify(l)}
                </td>
                <td>${a}</td>
                <td>
                  <span class="entropy-pill ${d}">${h.toFixed(4)} nats</span>
                </td>
              </tr>
            `})}
        </tbody>
      </table>
    `}};u.styles=ht`
    :host {
      display: block;
      min-height: 100vh;
      --bg: #0b0f17;
      --panel: #131b2e;
      --panel-alt: #19233c;
      --border: #263457;
      --text: #e8eefb;
      --muted: #94a3b8;
      --accent: #38bdf8;
      --pass: #22c55e;
      --warn: #f59e0b;
      --fail: #ef4444;
      background: var(--bg);
      color: var(--text);
    }

    header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 14px 24px;
      background: var(--panel);
      border-bottom: 1px solid var(--border);
    }

    .brand {
      display: flex;
      align-items: center;
      gap: 12px;
      flex-wrap: wrap;
    }

    .brand h1 {
      margin: 0;
      font-size: 18px;
      font-weight: 700;
      letter-spacing: -0.02em;
    }

    .badge {
      font-size: 11px;
      padding: 3px 9px;
      border-radius: 999px;
      font-weight: 600;
      background: rgba(56, 189, 248, 0.15);
      color: var(--accent);
      border: 1px solid rgba(56, 189, 248, 0.35);
    }

    .status-pill {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 12px;
      padding: 6px 12px;
      border-radius: 999px;
      background: var(--panel-alt);
      border: 1px solid var(--border);
    }

    .dot {
      width: 8px;
      height: 8px;
      border-radius: 50%;
      background: var(--muted);
    }
    .dot.ok {
      background: var(--pass);
      box-shadow: 0 0 8px var(--pass);
    }
    .dot.warn {
      background: var(--warn);
      box-shadow: 0 0 8px var(--warn);
    }

    main {
      max-width: 1400px;
      margin: 0 auto;
      padding: 22px;
      display: grid;
      grid-template-columns: 1fr 1.15fr;
      gap: 22px;
    }

    @media (max-width: 980px) {
      main {
        grid-template-columns: 1fr;
      }
    }

    .card {
      background: var(--panel);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 18px;
    }

    .card h2 {
      margin: 0 0 12px 0;
      font-size: 14px;
      color: var(--accent);
      text-transform: uppercase;
      letter-spacing: 0.05em;
    }

    label {
      display: block;
      font-size: 12px;
      font-weight: 600;
      color: var(--muted);
      margin: 12px 0 5px;
    }

    select,
    input[type='text'],
    textarea {
      width: 100%;
      box-sizing: border-box;
      background: var(--bg);
      color: var(--text);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 9px 11px;
      font-size: 13px;
      font-family: inherit;
    }

    textarea {
      font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
      resize: vertical;
    }

    button.primary {
      margin-top: 16px;
      width: 100%;
      padding: 11px 16px;
      border: none;
      border-radius: 8px;
      background: linear-gradient(135deg, #0284c7, #38bdf8);
      color: #041019;
      font-weight: 700;
      font-size: 14px;
      cursor: pointer;
    }

    button.primary:disabled {
      opacity: 0.6;
      cursor: wait;
    }

    .slot-table {
      width: 100%;
      border-collapse: collapse;
      margin-top: 10px;
      font-size: 13px;
    }

    .slot-table th,
    .slot-table td {
      text-align: left;
      padding: 9px 10px;
      border-bottom: 1px solid var(--border);
    }

    .slot-table th {
      color: var(--muted);
      font-size: 11px;
      text-transform: uppercase;
    }

    .entropy-pill {
      display: inline-block;
      padding: 2px 8px;
      border-radius: 6px;
      font-weight: 600;
      font-size: 12px;
    }
    .entropy-low {
      background: rgba(34, 197, 94, 0.16);
      color: #4ade80;
    }
    .entropy-mid {
      background: rgba(245, 158, 11, 0.16);
      color: #fbbf24;
    }
    .entropy-high {
      background: rgba(239, 68, 68, 0.16);
      color: #f87171;
    }

    pre {
      background: var(--bg);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 12px;
      overflow-x: auto;
      font-size: 12px;
      color: #cbd5e1;
    }

    .canvas-wrap {
      position: relative;
      display: inline-block;
      max-width: 100%;
      margin-top: 12px;
    }

    .canvas-wrap img {
      max-width: 100%;
      border-radius: 8px;
      display: block;
    }

    .canvas-wrap svg {
      position: absolute;
      top: 0;
      left: 0;
      width: 100%;
      height: 100%;
      pointer-events: none;
    }
  `;f([g()],u.prototype,"catalog",2);f([g()],u.prototype,"selectedIdx",2);f([g()],u.prototype,"vars",2);f([g()],u.prototype,"rawTemplate",2);f([g()],u.prototype,"uploadedDataURI",2);f([g()],u.prototype,"status",2);f([g()],u.prototype,"loading",2);f([g()],u.prototype,"elapsedSec",2);f([g()],u.prototype,"result",2);f([g()],u.prototype,"errorMsg",2);u=f([Rt("dgem-studio")],u);
