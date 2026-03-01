package intent

type Runtime string

const (
	RuntimeKubernetes Runtime = "kubernetes"
	RuntimeKnative    Runtime = "knative"
	RuntimeECS        Runtime = "ecs"
	RuntimeWASM       Runtime = "wasm"
	RuntimeAzure      Runtime = "azure"
)

type Strategy string

const (
	StrategyRolling   Strategy = "Rolling"
	StrategyBlueGreen Strategy = "BlueGreen"
	StrategyCanary    Strategy = "Canary"
)
