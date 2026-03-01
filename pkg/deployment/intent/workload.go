package intent

// WorkloadIntent points to the concrete runtime object created
// (Deployment, Knative Service, ECS Service, etc)
type WorkloadIntent struct {
	APIVersion string
	Kind       string
	Name       string
	Namespace  string
}
