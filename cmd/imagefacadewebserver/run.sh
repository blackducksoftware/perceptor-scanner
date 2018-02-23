set -e

oc create ns bds-perceptor
oc create -f if-serviceaccount.yaml --namespace=bds-perceptor
# allows launching of privileged containers for Docker machine access
oc adm policy add-scc-to-user privileged system:serviceaccount:bds-perceptor:if-sa
oc create -f if.yaml --namespace=bds-perceptor
oc create -f route.yaml --namespace=bds-perceptor
