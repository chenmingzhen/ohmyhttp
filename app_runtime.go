package ohmyhttp

const (
	defaultMaxBodyBytes    = 32 << 20
	defaultMultipartMemory = 8 << 20
)

// appRuntime 保存 App 处理请求所需的稳定依赖和配置。
type appRuntime struct {
	binder                      *requestBinder
	validator                   *defaultValidator
	encoder                     *defaultEncoder
	errorMapper                 ErrorMapper
	endpointResponseTransformer EndpointResponseTransformer
	requestObserver             RequestObserver
	errorObserver               ErrorObserver
	responseObserver            ResponseObserver
	notFound                    ResponseFunc
	methodNotAllowed            MethodNotAllowedFunc
	panicMapper                 PanicMapper
	maxBodyBytes                int64
	environment                 Environment
	recoverPanics               bool
}

func newAppRuntime(options ...AppOption) *appRuntime {
	runtime := &appRuntime{
		binder:       newRequestBinder(defaultMultipartMemory),
		validator:    newDefaultValidator(),
		encoder:      newDefaultEncoder(),
		errorMapper:  defaultErrorMapper{},
		maxBodyBytes: defaultMaxBodyBytes,
		environment:  EnvProduction,
	}
	for _, option := range options {
		option(runtime)
	}
	return runtime
}
