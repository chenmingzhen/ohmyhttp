package ohmyhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type passwordValidationRequest struct {
	Password string `json:"password" validate:"required,min=8" msg:"请输入密码" msg_min:"密码至少需要 8 个字符"`
	Confirm  string `json:"confirm" msg:"确认密码格式错误"`
	Email    string `json:"email" validate:"omitempty,email"`
	Accounts []*struct {
		Password string `json:"password" validate:"required" msg_required:"账户密码不能为空"`
	} `json:"accounts" validate:"dive"`
	Labels []string `json:"labels" validate:"dive,required" msg_required:"标签不能为空"`
}

func (r *passwordValidationRequest) ValidateRequest(context.Context) error {
	if r.Confirm != "" && r.Password != r.Confirm {
		return NewValidationError(nil, []Violation{
			{Field: "confirm", Rule: "password_match", Message: "两次密码输入不一致"},
		})
	}
	return nil
}

func TestValidationMessageTags(t *testing.T) {
	app := New()
	app.Post("/password", func(context.Context, *passwordValidationRequest) (struct{}, error) {
		return struct{}{}, nil
	})

	for _, tc := range []struct {
		name string
		body string
		want map[string]string
	}{
		{
			name: "field fallback",
			body: `{}`,
			want: map[string]string{"field": "password", "rule": "required", "message": "请输入密码"},
		},
		{
			name: "rule override",
			body: `{"password":"abc"}`,
			want: map[string]string{"field": "password", "rule": "min", "parameter": "8", "message": "密码至少需要 8 个字符"},
		},
		{
			name: "nested element",
			body: `{"password":"long-enough","accounts":[{}]}`,
			want: map[string]string{"field": "accounts[0].password", "rule": "required", "message": "账户密码不能为空"},
		},
		{
			name: "scalar element",
			body: `{"password":"long-enough","labels":[""]}`,
			want: map[string]string{"field": "labels[0]", "rule": "required", "message": "标签不能为空"},
		},
		{
			name: "no message tag",
			body: `{"password":"long-enough","email":"invalid"}`,
			want: map[string]string{"field": "email", "rule": "email"},
		},
		{
			name: "request validator",
			body: `{"password":"long-enough","confirm":"different"}`,
			want: map[string]string{"field": "confirm", "rule": "password_match", "message": "两次密码输入不一致"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/password", strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			app.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("unexpected HTTP status %d: %s", recorder.Code, recorder.Body)
			}
			var response struct {
				Code    string              `json:"code"`
				Details []map[string]string `json:"details"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Code != "validation_failed" || !reflect.DeepEqual(response.Details, []map[string]string{tc.want}) {
				t.Errorf("unexpected validation response: %s, want detail %v", recorder.Body, tc.want)
			}
		})
	}
}

func TestValidationMessageNestedPaths(t *testing.T) {
	type profile struct {
		Name string `json:"name" validate:"required" msg:"请填写名称"`
	}
	request := &struct {
		profile
		Values map[string]any `json:"values" validate:"dive"`
	}{
		Values: map[string]any{"a.b][0]": &profile{}},
	}
	err := newDefaultValidator().Validate(t.Context(), request)
	if err == nil {
		t.Fatal("expected validation to fail")
	}
	got := normalizeValidationError(err).Details
	want := []Violation{
		{Field: "profile.name", Rule: "required", Message: "请填写名称"},
		{Field: "values[a.b][0]].name", Rule: "required", Message: "请填写名称"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestValidationMessageAmbiguousMapPath(t *testing.T) {
	request := struct {
		Items map[string]map[string]any `validate:"dive,dive"`
	}{Items: map[string]map[string]any{
		"a": {"b][c": struct {
			Name string `validate:"required" msg:"请填写商品名称"`
		}{}},
		"a][b": {"c": struct {
			Name string `validate:"required" msg:"请填写店铺名称"`
		}{}},
	}}

	err := newDefaultValidator().Validate(t.Context(), request)
	if err == nil {
		t.Fatal("expected validation to fail")
	}
	got := normalizeValidationError(err).Details
	want := Violation{Field: "Items[a][b][c].Name", Rule: "required"}
	if !reflect.DeepEqual(got, []Violation{want, want}) {
		t.Errorf("unexpected details for ambiguous paths: %#v", got)
	}
}
