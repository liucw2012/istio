podname=productpage-v1-8554d58bff-d7j8d
ns=default

istioctl proxy-config listener listener $podname $ns -o json > lds.json

istioctl proxy-config router listener $podname $ns -o json > rds.json

istioctl proxy-config cluster listener $podname $ns -o json > cds.json

istioctl proxy-config endpoint listener $podname $ns -o json > eds.json
