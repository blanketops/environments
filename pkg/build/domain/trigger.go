package domain

type TriggerContext struct {
	Type         string // manual | pull_request | push | schedule
	Ref          string // refs/pull/123/head, refs/heads/main
	SHA          string // commit SHA
	RetryAttempt string // on-failure | always | never
}
