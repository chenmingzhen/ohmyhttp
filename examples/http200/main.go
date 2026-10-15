package main

import (
	"context"
	"errors"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/chenmingzhen/ohmyhttp"
	httpserver "github.com/chenmingzhen/ohmyhttp/server"
)

type APIResponse struct {
	Code       int    `json:"code"`
	Data       any    `json:"data,omitempty"`
	Successful bool   `json:"successful"`
	Message    string `json:"message,omitempty"`
}

type BusinessError struct {
	code    int
	message string
}

var _ ohmyhttp.HTTPError = (*BusinessError)(nil)

func NewBusinessError(code int, message string) *BusinessError {
	return &BusinessError{code: code, message: message}
}

func (e *BusinessError) Error() string        { return e.message }
func (e *BusinessError) HTTPStatus() int      { return http.StatusUnprocessableEntity }
func (e *BusinessError) ErrorCode() string    { return strconv.Itoa(e.code) }
func (e *BusinessError) ErrorMessage() string { return e.message }
func (e *BusinessError) ErrorDetails() any    { return nil }

type APIResponseTransformer struct{}

var _ ohmyhttp.EndpointResponseTransformer = APIResponseTransformer{}

func (APIResponseTransformer) TransformEndpointResponse(
	_ context.Context,
	_ ohmyhttp.RouteInfo,
	response ohmyhttp.Response,
	err error,
) ohmyhttp.Response {
	if err == nil {
		response.Body = APIResponse{
			Code:       0,
			Data:       response.Body,
			Successful: true,
		}
		return response
	}

	result := APIResponse{
		Code:       response.Status,
		Data:       nil,
		Successful: false,
	}
	var businessErr *BusinessError
	if errors.As(err, &businessErr) {
		response.Status = http.StatusOK
		result.Code = businessErr.code
		result.Message = businessErr.Error()
	} else {
		result.Code = -1
		result.Message = "unknown error, rid: " + strconv.FormatInt(time.Now().UnixMilli(), 10)
	}
	response.Body = result
	return response
}

type CreateOrderRequest struct {
	Quantity float64 `json:"quantity" validate:"required,gt=0"`
}

type CreateOrderResponse struct {
	Quantity int `json:"quantity"`
}

func CreateOrder(
	_ context.Context,
	request *CreateOrderRequest,
) (*CreateOrderResponse, error) {
	if request.Quantity != math.Trunc(request.Quantity) {
		panic("quantity must be an integer")
	}
	quantity := int(request.Quantity)
	if quantity > 100 {
		return nil, NewBusinessError(10001, "order quantity exceeds purchase limit")
	}
	return &CreateOrderResponse{Quantity: quantity}, nil
}

// logServerError 记录 500 错误。时间戳写在日志正文里，堆栈只留在服务端。
func logServerError(_ context.Context, err error, response ohmyhttp.Response) {
	if response.Status < http.StatusInternalServerError {
		return
	}
	message := "time=" + time.Now().Format("2006-01-02T15:04:05.000Z0700") +
		" level=error status=" + strconv.Itoa(response.Status) +
		" err=" + strconv.Quote(err.Error())
	if body, ok := response.Body.(APIResponse); ok && body.Message != "" {
		message += " message=" + strconv.Quote(body.Message)
	}
	var panicErr *ohmyhttp.PanicError
	if errors.As(err, &panicErr) {
		if stack := panicErr.StackTrace(); stack != "" {
			message += "\n" + stack
		}
	}
	log.New(os.Stderr, "", 0).Print(message)
}

func main() {
	const address = "127.0.0.1:8081"
	log.Printf("HTTP 200 business response demo listening on http://%s", address)
	if err := httpserver.Run(newApp(), httpserver.WithAddress(address)); err != nil {
		log.Fatal(err)
	}
}

func newApp() *ohmyhttp.App {
	app := ohmyhttp.New(
		ohmyhttp.WithRecovery(),
		ohmyhttp.WithErrorObserver(logServerError),
		ohmyhttp.WithEndpointResponseTransformer(APIResponseTransformer{}),
	)
	app.Post("/orders", CreateOrder)
	return app
}
