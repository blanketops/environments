package intent

type ReconciliationStrategy string

const (
	ReconciliationImperative ReconciliationStrategy = "Imperative"
	ReconciliationKustomize  ReconciliationStrategy = "Kustomize"
	ReconciliationHelm       ReconciliationStrategy = "Helm"
)
