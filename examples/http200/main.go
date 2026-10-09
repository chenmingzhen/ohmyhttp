// Package main demonstrates an application-owned protocol that returns business
// failures with HTTP 200 while keeping the business result in the JSON body.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/chenmingzhen/ohmyhttp"
	httpserver "github.com/chenmingzhen/ohmyhttp/server"
)

type APIResponse struct {
	Code       int  `json:"code"`
	Data       any  `json:"data"`
	Successful bool `json:"successful"`
}

type BusinessError struct {
	code    int
	message string
}

var _ ohmyhttp.PublicError = (*BusinessError)(nil)

func NewBusinessError(code int, message string) *BusinessError {
	return &BusinessError{code: code, message: message}
}

func (e *BusinessError) Error() string         { return e.message }
func (e *BusinessError) HTTPStatus() int       { return http.StatusUnprocessableEntity }
func (e *BusinessError) ErrorCode() string     { return strconv.Itoa(e.code) }
func (e *BusinessError) PublicMessage() string { return e.message }
func (e *BusinessError) ErrorDetails() any     { return nil }

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
		Code:       response.Status * 100,
		Data:       nil,
		Successful: false,
	}
	var businessErr *BusinessError
	if errors.As(err, &businessErr) {
		response.Status = http.StatusOK
		result.Code = businessErr.code
	}
	response.Body = result
	return response
}

type CreateOrderRequest struct {
	Quantity int `json:"quantity" validate:"required,gt=0"`
}

type CreateOrderResponse struct {
	Quantity int `json:"quantity"`
}

func CreateOrder(
	_ context.Context,
	request *CreateOrderRequest,
) (*CreateOrderResponse, error) {
	if request.Quantity > 100 {
		return nil, NewBusinessError(10001, "order quantity exceeds purchase limit")
	}
	return &CreateOrderResponse{Quantity: request.Quantity}, nil
}

func main() {
	app := ohmyhttp.New(
		ohmyhttp.WithEndpointResponseTransformer(APIResponseTransformer{}),
	)
	app.Post("/orders", CreateOrder)

	const address = "127.0.0.1:8081"
	log.Printf("HTTP 200 business response demo listening on http://%s", address)
	if err := httpserver.Run(app, httpserver.WithAddress(address)); err != nil {
		log.Fatal(err)
	}
}
