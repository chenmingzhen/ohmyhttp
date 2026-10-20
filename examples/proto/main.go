// proto 示例展示由 .proto 生成的 protobuf message 的 JSON 请求绑定、校验
// 和响应编码。
//
// 运行：
//
//	cd examples/proto && go run .
//
// 请求：
//
//	curl -X POST http://127.0.0.1:8082/echo \
//	  -H 'Content-Type: application/json' \
//	  -d '{"name":"Ada","message":"hello"}'
package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	"buf.build/go/protovalidate"
	"github.com/chenmingzhen/ohmyhttp"
	"github.com/chenmingzhen/ohmyhttp/examples/proto/gen/demopb"
	httpserver "github.com/chenmingzhen/ohmyhttp/server"
	"google.golang.org/protobuf/proto"
)

func Echo(
	_ context.Context,
	request *demopb.EchoRequest,
) (*demopb.EchoResponse, error) {
	return &demopb.EchoResponse{
		Greeting: "hello, " + request.GetName(),
		Message:  request.GetMessage(),
	}, nil
}

func validateProtoRequest(
	_ context.Context,
	message proto.Message,
) error {
	err := protovalidate.Validate(message)
	if err == nil {
		return nil
	}

	var validationErr *protovalidate.ValidationError
	if !errors.As(err, &validationErr) {
		return err
	}

	details := make([]ohmyhttp.Violation, 0, len(validationErr.Violations))
	for _, violation := range validationErr.Violations {
		details = append(details, ohmyhttp.Violation{
			Field:   protovalidate.FieldPathString(violation.Proto.GetField()),
			Rule:    violation.Proto.GetRuleId(),
			Message: violation.String(),
		})
	}

	return ohmyhttp.NewValidationError(err, details)
}

func newApp() *ohmyhttp.App {
	// WithDiscardUnknownFields 只影响请求解码。未调用 WithProtoJSON 时，
	// 响应使用 protojson.MarshalOptions{UseProtoNames: true}，并省略未赋值字段。
	// 要写出零值且仍使用 proto 字段名时：
	// ohmyhttp.WithProtoJSON(protojson.MarshalOptions{
	//     UseProtoNames: true, EmitUnpopulated: true,
	// })
	app := ohmyhttp.New(
		ohmyhttp.WithProtoValidator(validateProtoRequest),
		ohmyhttp.WithDiscardUnknownFields(true),
	)
	app.Post("/echo", Echo)
	return app
}

func main() {
	const address = "127.0.0.1:8082"
	log.Printf("protobuf demo listening on http://%s", address)
	if err := httpserver.Run(newApp(), httpserver.WithAddress(address)); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}
}
