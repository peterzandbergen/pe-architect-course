# CapStone Assignment

## Checks

These are the running pods

```
# k get po -A
NAMESPACE              NAME                                                     READY   STATUS    RESTARTS      AGE
argo-rollouts          argo-rollouts-79b89d8856-7rtth                           1/1     Running   0             3d23h
default                rollouts-demo-5b74cfbf86-2rn6w                           1/1     Running   0             5d3h
default                rollouts-demo-5b74cfbf86-9nd8l                           1/1     Running   0             5d3h
default                rollouts-demo-5b74cfbf86-chn8v                           1/1     Running   0             5d3h
default                rollouts-demo-5b74cfbf86-f5nlh                           1/1     Running   0             5d3h
default                rollouts-demo-5b74cfbf86-rqdv5                           1/1     Running   0             5d3h
engineering-platform   teams-operator-5c4fcc9957-g8mx2                          1/1     Running   0             5d5h
engineering-platform   teams-ui-54556bfbc9-7rzfp                                1/1     Running   0             5d5h
engineering-platform   teams-ui-54556bfbc9-96hml                                1/1     Running   0             5d5h
engineering-platform   teams-ui-54556bfbc9-c2zcl                                1/1     Running   0             5d5h
falco-system           falco-f8dfx                                              2/2     Running   0             16d
falco-system           falco-falcosidekick-6cc79bf686-8nsrl                     1/1     Running   0             16d
falco-system           falco-falcosidekick-6cc79bf686-bbfvr                     1/1     Running   0             16d
gatekeeper-system      gatekeeper-audit-5c8b464f5-mcbjh                         1/1     Running   2 (20d ago)   22d
gatekeeper-system      gatekeeper-controller-manager-b558c4d77-h89k4            1/1     Running   1 (20d ago)   22d
gatekeeper-system      gatekeeper-controller-manager-b558c4d77-nf8bx            1/1     Running   1 (20d ago)   22d
gatekeeper-system      gatekeeper-controller-manager-b558c4d77-vwd56            1/1     Running   1 (20d ago)   22d
ingress-nginx          ingress-nginx-controller-694bf66bcf-b7997                1/1     Running   1 (20d ago)   22d
keycloak               keycloak-9fdcbc4d9-nchgz                                 1/1     Running   0             5d6h
keycloak               keycloak-postgres-5cdc44d556-62fn5                       1/1     Running   0             5d6h
kube-system            coredns-668d6bf9bc-5xnk6                                 1/1     Running   1 (20d ago)   22d
kube-system            coredns-668d6bf9bc-tpvsp                                 1/1     Running   1 (20d ago)   22d
kube-system            etcd-5min-idp-control-plane                              1/1     Running   1 (20d ago)   22d
kube-system            kindnet-4q4zk                                            1/1     Running   1 (20d ago)   22d
kube-system            kube-apiserver-5min-idp-control-plane                    1/1     Running   1 (20d ago)   22d
kube-system            kube-controller-manager-5min-idp-control-plane           1/1     Running   1 (20d ago)   22d
kube-system            kube-proxy-zdxp9                                         1/1     Running   1 (20d ago)   22d
kube-system            kube-scheduler-5min-idp-control-plane                    1/1     Running   1 (20d ago)   22d
kube-system            metrics-server-5dd7b49d79-4p969                          1/1     Running   2 (20d ago)   22d
local-path-storage     local-path-provisioner-58cc7856b6-trzjs                  1/1     Running   2 (20d ago)   22d
monitoring             alertmanager-grafana-stack-kube-prometh-alertmanager-0   2/2     Running   0             5d8h
monitoring             grafana-stack-67b8f5cbdc-p2xdh                           3/3     Running   0             5d8h
monitoring             grafana-stack-kube-prometh-operator-766658bfd4-628ht     1/1     Running   0             5d8h
monitoring             grafana-stack-kube-state-metrics-5df8b94ff-w2jnd         1/1     Running   0             5d8h
monitoring             grafana-stack-prometheus-node-exporter-42trs             1/1     Running   0             5d8h
monitoring             prometheus-grafana-stack-kube-prometh-prometheus-0       2/2     Running   0             5d8h
my-api                 my-api-7bc79457dd-p9mrh                                  1/1     Running   0             5d2h
teams-api              teams-api-754c855cd7-ns7zd                               1/1     Running   0             5d5h
```

## Argo Rollouts

Argo Rollouts deployed and running. 

I use coder port forward and ingress with sslip.io domains to get to the services.

With the Krew Plugin installed and the dashboard command running I can see the Argo Rollout dashboard in the browser.

```
NAMESPACE   NAME            DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
default     rollouts-demo   5         5         5            5           5d4h
```

The plugin is installed and shows the rollouts.

```
# k argo rollouts list rollouts
NAME           STRATEGY   STATUS        STEP  SET-WEIGHT  READY  DESIRED  UP-TO-DATE  AVAILABLE
rollouts-demo  Canary     Healthy       8/8   100         5/5    5        5           5
```

BTW, I use the coder port-forward command to access the services on my laptop. 

Instead of port forwards I use ingresses and sslip.io names.

```
NAMESPACE              NAME                         CLASS   HOSTS                                   ADDRESS       PORTS   AGE
default                my-api                       nginx   rollouts-demo.127.0.0.1.sslip.io        10.96.41.91   80      5d3h
engineering-platform   teams-operator-api-ingress   nginx   teams-operator-api.127.0.0.1.sslip.io   10.96.41.91   80      5d6h
engineering-platform   teams-ui-ingress             nginx   teams-ui.127.0.0.1.sslip.io             10.96.41.91   80      5d6h
keycloak               keycloak-ingress             nginx   platform-auth.127.0.0.1.sslip.io        10.96.41.91   80      5d7h
monitoring             grafana-ingress              nginx   grafana.127.0.0.1.sslip.io              10.96.41.91   80      5d9h
my-api                 my-api                       nginx   my-api.127.0.0.1.sslip.io               10.96.41.91   80      5d4h
teams-api              teams-api-ingress            nginx   teams-api.127.0.0.1.sslip.io            10.96.41.91   80      5d6h
```

