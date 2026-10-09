package ohmyhttp

func responseFromResult(result any, successStatus int) Response {
	switch value := result.(type) {
	case Response:
		if value.Status == 0 {
			value.Status = successStatus
		}
		return value
	case *Response:
		if value == nil {
			return Response{Status: successStatus}
		}
		response := *value
		if response.Status == 0 {
			response.Status = successStatus
		}
		return response
	default:
		return Response{Status: successStatus, Body: result}
	}
}
