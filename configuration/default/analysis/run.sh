podname=productpage-v1-8554d58bff-d7j8d
ns=default

rm lds.json rds.json cds.json eds.json sds.json

istioctl proxy-config listener $podname -n $ns -o json > lds.json

istioctl proxy-config route $podname -n $ns -o json > rds.json

istioctl proxy-config cluster $podname -n $ns -o json > cds.json

istioctl proxy-config endpoint $podname -n $ns -o json > eds.json

istioctl proxy-config secret $podname -n $ns -o json > sds.json