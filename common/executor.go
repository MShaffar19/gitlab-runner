package common

import (
	log "github.com/Sirupsen/logrus"
)

type Executor interface {
	Prepare(globalConfig *Config, config *RunnerConfig, build *Build) error
	Start() error
	Wait() error
	Finish(err error)
	Cleanup()
}

type ExecutorProvider interface {
	CanCreate() bool
	Create() Executor
	Features() *FeaturesInfo
}

var executors map[string]ExecutorProvider

func RegisterExecutor(executor string, provider ExecutorProvider) {
	log.Debugln("Registering", executor, "executor...")

	if executors == nil {
		executors = make(map[string]ExecutorProvider)
	}
	if _, ok := executors[executor]; ok {
		panic("Executor already exist: " + executor)
	}
	executors[executor] = provider
}

func GetExecutorProvider(executor string) ExecutorProvider {
	if executors == nil {
		return nil
	}

	provider, _ := executors[executor]
	return provider
}

func GetExecutorFeatures(executor string) *FeaturesInfo {
	provider := GetExecutorProvider(executor)
	if provider != nil {
		return provider.Features()
	}
	return nil
}

func NewExecutor(executor string) Executor {
	provider := GetExecutorProvider(executor)
	if provider != nil {
		return provider.Create()
	}

	return nil
}

func GetExecutors() []string {
	names := []string{}
	if executors != nil {
		for name := range executors {
			names = append(names, name)
		}
	}
	return names
}
