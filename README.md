# Kubevirt UI Proxy

Kubevirt UI proxy data pod resolve the performance issue in large scale cluster by implementing filtering and pagination on data from kube api server in BE instead of UI.

## Usage

### Dev mode deployment

Save this YAML to a file locally:

```yaml
---
kind: Deployment
apiVersion: apps/v1
metadata:
  name: kubevirt-proxy-pod
  namespace: openshift-cnv
  labels:
    app: kubevirt-proxy-pod
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kubevirt-proxy-pod
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: kubevirt-proxy-pod
    spec:
      containers:
        - name: kubevirt-proxy-pod
          image: "quay.io/mschatzm/kubevirt-proxy-pod:test"
          ports:
            - containerPort: 8080
              protocol: TCP
          imagePullPolicy: Always
          volumeMounts:
            - name: cert
              mountPath: "/app/cert"
              readOnly: true
      volumes:
        - name: cert
          secret:
            secretName: kubevirt-proxy-cert
            optional: true
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 25%
      maxSurge: 25%

---
kind: Service
apiVersion: v1
metadata:
  name: kubevirt-proxy-pod
  namespace: openshift-cnv
  annotations:
    service.beta.openshift.io/serving-cert-secret-name: kubevirt-proxy-cert
spec:
  ipFamilies:
    - IPv4
  ports:
    - protocol: TCP
      port: 80
      targetPort: 8080
  selector:
    app: kubevirt-proxy-pod

---
kind: Route
apiVersion: route.openshift.io/v1
metadata:
  name: kubevirt-proxy-pod
  namespace: openshift-cnv
  annotations:
    haproxy.router.openshift.io/hsts_header: max-age=31536000;includeSubDomains;preload
spec:
  host: $HOST #example: kubevirt-proxy-pod-openshift-cnv.apps.uit-413-0602.rhos-psi.cnv-qe.rhood.us
  to:
    kind: Service
    name: kubevirt-proxy-pod
    weight: 100
  port:
    targetPort: 8080
  tls:
    termination: reencrypt
  wildcardPolicy: None
```

This yaml will create 3 resources.

1. Deployment (pod where the proxy will run)
2. Service
3. Route (in production mode this will not be created - instead kubevirt-plugin will be added to include a proxy so this route is not needed)

