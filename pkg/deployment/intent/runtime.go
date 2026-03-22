package intent

type Runtime string

const (
	RuntimeKubernetes Runtime = "kubernetes.io/container-runtime"
	RuntimeKnative    Runtime = "knative.dev/service-runtime"
	RuntimeWASM       Runtime = "blanketops.dev/wasm-runtime"
	RuntimeECS        Runtime = "blanketops.dev/aws-ecs"
	RuntimeAzure      Runtime = "blanketops.dev/azure-container"
)

type Strategy string

const (
	StrategyRolling   Strategy = "Rolling"
	StrategyBlueGreen Strategy = "BlueGreen"
	StrategyCanary    Strategy = "Canary"
)
