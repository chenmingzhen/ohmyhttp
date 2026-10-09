package ohmyhttp

import (
	"fmt"
	"os"
	"strings"
)

// Environment 表示应用当前运行环境。
//
// ohmyhttp 会基于环境调整开发期诊断行为。
type Environment string

const (
	// EnvLocal 表示本地开发环境。
	EnvLocal Environment = "local"
	// EnvDevelopment 表示开发环境。
	EnvDevelopment Environment = "development"
	// EnvTest 表示测试环境。
	EnvTest Environment = "test"
	// EnvStaging 表示预发布环境。
	EnvStaging Environment = "staging"
	// EnvProduction 表示生产环境，也是默认环境。
	EnvProduction Environment = "production"
)

// WithEnvironment 设置应用运行环境。
//
// 默认环境是 EnvProduction。local、development 和 test 会启用开发期提示；
// staging 和 production 会保持安静。
func WithEnvironment(environment Environment) AppOption {
	return func(runtime *appRuntime) {
		runtime.environment = normalizeEnvironment(environment)
	}
}

// WithEnvironmentFromEnv 从环境变量读取应用运行环境。
//
// 默认依次读取 OHMYHTTP_ENV、APP_ENV、GO_ENV 和 ENV。都为空时使用 EnvProduction。
func WithEnvironmentFromEnv(keys ...string) AppOption {
	return func(runtime *appRuntime) {
		if len(keys) == 0 {
			keys = []string{"OHMYHTTP_ENV", "APP_ENV", "GO_ENV", "ENV"}
		}
		for _, key := range keys {
			if value := strings.TrimSpace(os.Getenv(key)); value != "" {
				runtime.environment = normalizeEnvironment(Environment(value))
				return
			}
		}
		runtime.environment = EnvProduction
	}
}

// String 返回规范化后的环境名称。
func (environment Environment) String() string {
	return string(normalizeEnvironment(environment))
}

// IsDevelopment 表示当前环境是否属于本地、开发或测试环境。
func (environment Environment) IsDevelopment() bool {
	return isDevelopmentEnvironment(environment)
}

var _ fmt.Stringer = Environment("")

func normalizeEnvironment(environment Environment) Environment {
	value := strings.ToLower(strings.TrimSpace(string(environment)))
	switch value {
	case "local":
		return EnvLocal
	case "dev", "develop", "development":
		return EnvDevelopment
	case "test", "testing":
		return EnvTest
	case "stage", "staging":
		return EnvStaging
	case "", "prod", "production", "release":
		return EnvProduction
	default:
		return Environment(value)
	}
}

func isDevelopmentEnvironment(environment Environment) bool {
	switch normalizeEnvironment(environment) {
	case EnvLocal, EnvDevelopment, EnvTest:
		return true
	default:
		return false
	}
}
