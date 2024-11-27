# Discover Workload

An command line interface for extracting certifiable components from an
installed instance of a product or application.

This tool allows application developers to install an Operator or Helm Chart
they intend to certify with Red Hat, and auto-detect the components that must be
certified.

## Note:

This project is in early development. Use at your own risk.

## Quick Start

1. Deploy your application to an OpenShift cluster.
2. Once the application is deployed and you're happy with it, run `discover-workload` against the cluster. E.g.:


```shell
./discover-workload \
    --kubeconfig /path/to/kubeconfig \
    --duration 2m \
    --log-level DEBUG \
    --selector "my.example.com/app=my-app"
        check-this-ns \
        also-this-ns \
        also-here
```

3. Review the manifest produced for any workload components that are invalid, and modify as needed.