# Helm Plugin References

## Documentation

### Plugin Documentation
- [Helm v2-alpha Plugin Documentation](../../../../docs/book/src/plugins/available/helm-v2-alpha.md)

### Helm Resources
- [Helm Chart Best Practices](https://helm.sh/docs/chart_best_practices/)
- [Helm Documentation](https://helm.sh/docs/)
- [Helm Best Practices - Custom Resource Definitions](https://helm.sh/docs/chart_best_practices/custom_resource_definitions/)
- [Helm Best Practices - Values](https://helm.sh/docs/chart_best_practices/values/)
- [Helm Best Practices - Templates](https://helm.sh/docs/chart_best_practices/templates/)

### Kubernetes Resources
- [Kubernetes Operator Pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
- [Kubernetes Documentation Style Guide](https://kubernetes.io/docs/contribute/style/style-guide/)
- [Kubernetes API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)

### Code References
- Plugin source: `pkg/plugins/optional/helm/v2alpha/`
- Values generation: `pkg/plugins/optional/helm/v2alpha/scaffolds/internal/templates/values.go`
- Feature extraction: `pkg/plugins/optional/helm/v2alpha/scaffolds/internal/extractor/`

## Key Make Targets

```bash
make install           # Install updated kubebuilder binary
make generate-charts   # Regenerate sample charts in testdata
make verify-helm       # Validate all Helm charts (yamllint + helm lint + kube-linter)
```

## Chart Locations

Sample charts are located in:
- `testdata/project-v4-with-plugins/dist/chart/`
- `docs/book/src/getting-started/testdata/project/dist/chart/`
- `docs/book/src/cronjob-tutorial/testdata/project/dist/chart/`
- `docs/book/src/multiversion-tutorial/testdata/project/dist/chart/`
