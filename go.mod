module github.com/openshift/cluster-nfd-operator

go 1.16

require (
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	github.com/openshift/api v3.9.0+incompatible
	github.com/openshift/client-go v3.9.0+incompatible
	github.com/openshift/custom-resource-status v0.0.0-20210221154447-420d9ecf2a00
	github.com/prometheus/client_golang v1.11.0
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	k8s.io/klog/v2 v2.9.0
	k8s.io/kubectl v0.22.2
	sigs.k8s.io/controller-runtime v0.10.2
)
