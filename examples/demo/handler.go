package demo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/chenmingzhen/ohmyhttp"
	"github.com/chenmingzhen/ohmyhttp/errorx"
)

type LoginRequest struct {
	TenantID int    `path:"tenant_id" validate:"required,gt=0"`
	Account  string `                 validate:"required"       json:"account"`
	Password string `                 validate:"required,min=8" json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type OAuthTokenRequest struct {
	GrantType    string `form:"grant_type"    validate:"required,oneof=password refresh_token" doc:"授权类型"`
	Account      string `form:"account"                                                        doc:"password 模式使用的账号"`
	Password     string `form:"password"                                                       doc:"password 模式使用的密码"`
	RefreshToken string `form:"refresh_token"                                                  doc:"refresh_token 模式使用的刷新令牌"`
}

func (r *OAuthTokenRequest) ValidateRequest(_ context.Context) error {
	switch r.GrantType {
	case "password":
		if r.Account == "" || r.Password == "" {
			return ohmyhttp.NewValidationError(
				fmt.Errorf("account and password are required for password grant"),
				[]ohmyhttp.Violation{
					{
						Field:   "account",
						Rule:    "required_for_password_grant",
						Message: "account is required when grant_type is password",
					},
					{
						Field:   "password",
						Rule:    "required_for_password_grant",
						Message: "password is required when grant_type is password",
					},
				},
			)
		}
	case "refresh_token":
		if r.RefreshToken == "" {
			return ohmyhttp.NewValidationError(
				fmt.Errorf("refresh_token is required for refresh_token grant"),
				[]ohmyhttp.Violation{
					{
						Field:   "refresh_token",
						Rule:    "required_for_refresh_grant",
						Message: "refresh_token is required when grant_type is refresh_token",
					},
				},
			)
		}
	}
	return nil
}

type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type CreateSessionRequest struct {
	Account  string `json:"account"  validate:"required"       doc:"登录账号"`
	Password string `json:"password" validate:"required,min=8" doc:"登录密码"`
}

type CreateSessionResponse struct {
	Token string `json:"token" doc:"访问令牌"`
}

type ProfileRequest struct {
	Page    int  `query:"page"    validate:"required,gte=1,lte=100"`
	Verbose bool `query:"verbose"`
}

type ProfileResponse struct {
	User    string `json:"user"`
	Role    string `json:"role"`
	Page    int    `json:"page"`
	Verbose bool   `json:"verbose"`
}

type UpdateProfileRequest struct {
	UserID  string             `header:"X-User-ID" validate:"required" doc:"当前用户 ID"`
	Profile *UpdateProfileBody `                   validate:"required" doc:"要更新的用户资料" json:"profile"`
}

type UpdateProfileBody struct {
	DisplayName string  `json:"display_name"  validate:"required,min=2,max=40" doc:"展示名称"      example:"Demo User"`
	Bio         *string `json:"bio,omitempty" validate:"omitempty,max=120"     doc:"个人简介，可以为空"                     nullable:"true"`
}

type UpdateProfileResponse struct {
	UserID      string  `json:"user_id"`
	DisplayName string  `json:"display_name"`
	Bio         *string `json:"bio,omitempty"`
}

type AdminRequest struct{}

type AdminResponse struct {
	Message string `json:"message"`
}

type EmptyRequest struct{}

type CreateReportRequest struct {
	TenantID      int    `path:"tenant_id" validate:"required,gt=0"`
	ClientVersion string `                 validate:"required"                    header:"X-Client-Version"`
	Name          string `                 validate:"required"                                              json:"name"`
	Format        string `                 validate:"required,oneof=csv pdf xlsx"                           json:"format"`
}

type CreateReportResponse struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	ClientVersion string `json:"client_version"`
}

type CreateUserRequest struct {
	Username        string `json:"username"         validate:"required,username"`
	Email           string `json:"email"            validate:"required,email"`
	Password        string `json:"password"         validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required"`
}

func (r *CreateUserRequest) ValidateRequest(_ context.Context) error {
	if r.Password != r.ConfirmPassword {
		return ohmyhttp.NewValidationError(
			fmt.Errorf("password and confirm_password do not match"),
			[]ohmyhttp.Violation{
				{
					Field:   "confirm_password",
					Rule:    "password_match",
					Message: "password and confirm_password must match",
				},
			},
		)
	}
	return nil
}

type CreateUserResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type UploadAvatarRequest struct {
	UserID      string                `path:"user_id" validate:"required"`
	DisplayName string                `               validate:"required" form:"display_name"`
	Avatar      *multipart.FileHeader `               validate:"required" form:"avatar"`
}

type UploadAvatarResponse struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	FileName    string `json:"file_name"`
	FileSize    int64  `json:"file_size"`
}

type ReportID string

type ReportRequest struct {
	ReportID ReportID `path:"report_id" validate:"required"`
}

type SearchReportsRequest struct {
	Status string   `query:"status" validate:"omitempty,oneof=queued done failed" default:"queued" doc:"报表状态"`
	Tags   []string `query:"tag"    validate:"omitempty,dive,required"                             doc:"可重复 query 参数，例如 ?tag=finance&tag=monthly"`
	Page   int      `query:"page"   validate:"omitempty,gte=1,lte=100"            default:"1"      doc:"页码"`
}

type ReportSummary struct {
	ID     ReportID `json:"id"`
	Name   string   `json:"name"`
	Status string   `json:"status"`
	Tags   []string `json:"tags,omitempty"`
}

type SearchReportsResponse struct {
	Page  int             `json:"page"`
	Total int             `json:"total"`
	Items []ReportSummary `json:"items"`
}

type ReportMetadataResponse struct {
	ID        ReportID  `json:"id"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ExportReportRequest struct {
	ReportID ReportID `path:"report_id" validate:"required"`
	Rows     int      `                 validate:"required,gte=1,lte=10" query:"rows"`
}

type EventsRequest struct {
	Count int `query:"count" validate:"omitempty,gte=1,lte=5"`
}

type Handler struct{}

type PongResponse struct {
	Value string `json:"value"`
}

func (*Handler) Home(
	_ context.Context,
	_ *EmptyRequest,
) (ohmyhttp.Response, error) {
	return ohmyhttp.HTML(
		http.StatusOK,
		`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <title>ohmyhttp</title>
</head>
<body>
  <h1>ohmyhttp</h1>
  <p>The demo server is running.</p>
</body>
</html>`,
	), nil
}

func (*Handler) Ping(
	ctx context.Context,
	_ *EmptyRequest,
) (PongResponse, error) {
	return PongResponse{Value: "pong"}, nil
}

func (*Handler) HelloHTML(
	_ context.Context,
	_ *EmptyRequest,
) (ohmyhttp.Response, error) {
	return ohmyhttp.HTML(
		http.StatusOK,
		`<!doctype html><html><body><h1>ohmyhttp</h1><p>Hello from HTML response.</p></body></html>`,
	), nil
}

func (*Handler) RawBytes(
	_ context.Context,
	_ *EmptyRequest,
) (ohmyhttp.Response, error) {
	return ohmyhttp.Bytes(
		http.StatusOK,
		"application/octet-stream",
		[]byte("raw bytes from ohmyhttp\n"),
	), nil
}

func (*Handler) Login(
	_ context.Context,
	request *LoginRequest,
) (*LoginResponse, error) {
	roles := map[string]string{
		"demo":     "member",
		"admin":    "admin",
		"disabled": "member",
	}
	role, exists := roles[request.Account]
	if !exists || request.Password != "password" {
		return nil, ohmyhttp.NewError(
			http.StatusUnauthorized,
			"account or password is invalid",
			ohmyhttp.WithCode("invalid_credentials"),
		)
	}
	token, err := issueDemoToken(User{Name: request.Account, Role: role}, 2*time.Hour)
	if err != nil {
		return nil, errorx.WithStack(err)
	}
	return &LoginResponse{Token: token}, nil
}

func (*Handler) IssueOAuthToken(
	_ context.Context,
	request *OAuthTokenRequest,
) (*OAuthTokenResponse, error) {
	if request.GrantType == "password" &&
		(request.Account != "demo" || request.Password != "password") {
		return nil, ohmyhttp.NewError(
			http.StatusUnauthorized,
			"account or password is invalid",
			ohmyhttp.WithCode("invalid_credentials"),
		)
	}
	if request.GrantType == "refresh_token" && request.RefreshToken != "refresh-token" {
		return nil, ohmyhttp.NewError(
			http.StatusUnauthorized,
			"refresh token is invalid",
			ohmyhttp.WithCode("invalid_refresh_token"),
		)
	}
	return &OAuthTokenResponse{
		AccessToken:  "example-access-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: "refresh-token",
	}, nil
}

func (*Handler) CreateSession(
	ctx context.Context,
	request *CreateSessionRequest,
) (ohmyhttp.Response, error) {
	if request.Account != "demo" || request.Password != "password" {
		return ohmyhttp.Response{}, ohmyhttp.NewError(
			http.StatusUnauthorized,
			"account or password is invalid",
			ohmyhttp.WithCode("invalid_credentials"),
		)
	}
	return ohmyhttp.JSON(
		http.StatusCreated,
		CreateSessionResponse{Token: "example-token"},
	).
		WithHeader("X-Request-ID", RequestIDFromContext(ctx)).
		WithCookie(&http.Cookie{
			Name:     "demo_session",
			Value:    "example-token",
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		}), nil
}

func (*Handler) UpdateProfile(
	_ context.Context,
	request *UpdateProfileRequest,
) (*UpdateProfileResponse, error) {
	return &UpdateProfileResponse{
		UserID:      request.UserID,
		DisplayName: request.Profile.DisplayName,
		Bio:         request.Profile.Bio,
	}, nil
}

func (*Handler) CreateReport(
	_ context.Context,
	request *CreateReportRequest,
) (*CreateReportResponse, error) {
	return &CreateReportResponse{
		ID:            "monthly",
		Status:        "queued",
		ClientVersion: request.ClientVersion,
	}, nil
}

func (*Handler) SearchReports(
	_ context.Context,
	request *SearchReportsRequest,
) (*SearchReportsResponse, error) {
	page := request.Page
	if page == 0 {
		page = 1
	}
	status := request.Status
	if status == "" {
		status = "queued"
	}
	items := []ReportSummary{
		{
			ID:     "monthly",
			Name:   "Monthly report",
			Status: status,
			Tags:   request.Tags,
		},
	}
	return &SearchReportsResponse{
		Page:  page,
		Total: len(items),
		Items: items,
	}, nil
}

func (*Handler) CreateUser(
	_ context.Context,
	request *CreateUserRequest,
) (*CreateUserResponse, error) {
	return &CreateUserResponse{
		ID:       "user-1",
		Username: request.Username,
	}, nil
}

func (*Handler) UploadAvatar(
	_ context.Context,
	request *UploadAvatarRequest,
) (*UploadAvatarResponse, error) {
	return &UploadAvatarResponse{
		UserID:      request.UserID,
		DisplayName: request.DisplayName,
		FileName:    request.Avatar.Filename,
		FileSize:    request.Avatar.Size,
	}, nil
}

func (*Handler) Profile(
	ctx context.Context,
	request *ProfileRequest,
) (*ProfileResponse, error) {
	user := UserFromContext(ctx)
	return &ProfileResponse{
		User:    user.Name,
		Role:    user.Role,
		Page:    request.Page,
		Verbose: request.Verbose,
	}, nil
}

func (*Handler) Admin(
	ctx context.Context,
	_ *AdminRequest,
) (*AdminResponse, error) {
	if UserFromContext(ctx).Role != "admin" {
		return nil, ohmyhttp.NewError(
			http.StatusForbidden,
			"admin role is required",
			ohmyhttp.WithCode("permission_denied"),
		)
	}
	return &AdminResponse{Message: "welcome, admin"}, nil
}

func (*Handler) ReportMetadata(
	_ context.Context,
	request *ReportRequest,
) (ohmyhttp.Response, error) {
	if err := checkReportAccess(request.ReportID); err != nil {
		return ohmyhttp.Response{}, err
	}
	return ohmyhttp.JSON(http.StatusOK, ReportMetadataResponse{
		ID:        request.ReportID,
		Version:   3,
		UpdatedAt: time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC),
	}).
		WithHeader("ETag", fmt.Sprintf(`W/"report-%s-v3"`, request.ReportID)).
		WithHeader("Cache-Control", "max-age=60"), nil
}

func (*Handler) DownloadReport(
	_ context.Context,
	request *ReportRequest,
) (ohmyhttp.Response, error) {
	if err := checkReportAccess(request.ReportID); err != nil {
		return ohmyhttp.Response{}, err
	}
	reportPath := sampleReportPath()
	return ohmyhttp.Attachment(
		os.DirFS(filepath.Dir(reportPath)),
		filepath.Base(reportPath),
		"monthly-report.txt",
	), nil
}

func (*Handler) RedirectReport(
	_ context.Context,
	request *ReportRequest,
) (ohmyhttp.Response, error) {
	if err := checkReportAccess(request.ReportID); err != nil {
		return ohmyhttp.Response{}, err
	}
	return ohmyhttp.Redirect(http.StatusFound, "/reports/monthly/download"), nil
}

func (*Handler) ExportReportCSV(
	_ context.Context,
	request *ExportReportRequest,
) (ohmyhttp.Response, error) {
	if err := checkReportAccess(request.ReportID); err != nil {
		return ohmyhttp.Response{}, err
	}

	response := ohmyhttp.Streaming(
		http.StatusOK,
		"text/csv; charset=utf-8",
		func(ctx context.Context, writer io.Writer, flush func() error) error {
			if _, err := io.WriteString(writer, "id,name\n"); err != nil {
				return err
			}
			if err := flush(); err != nil {
				return err
			}

			for row := 1; row <= request.Rows; row++ {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
				if _, err := fmt.Fprintf(writer, "%d,report-row-%d\n", row, row); err != nil {
					return err
				}
				if err := flush(); err != nil {
					return err
				}
				time.Sleep(150 * time.Millisecond)
			}
			return nil
		},
	)

	response.Header = http.Header{
		"Content-Disposition": {`attachment; filename="monthly-export.csv"`},
	}
	return response, nil
}

func (*Handler) Events(
	_ context.Context,
	request *EventsRequest,
) (ohmyhttp.Response, error) {
	count := request.Count
	if count == 0 {
		count = 3
	}

	return ohmyhttp.EventStream(
		func(ctx context.Context, writer io.Writer, flush func() error) error {
			for index := 1; index <= count; index++ {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
				if _, err := fmt.Fprintf(writer, "event: tick\ndata: %d\n\n", index); err != nil {
					return err
				}
				if err := flush(); err != nil {
					return err
				}
				time.Sleep(time.Second)
			}
			return nil
		},
	), nil
}

func (*Handler) DeleteReport(
	_ context.Context,
	request *ReportRequest,
) (ohmyhttp.Response, error) {
	if err := checkReportAccess(request.ReportID); err != nil {
		return ohmyhttp.Response{}, err
	}
	return ohmyhttp.NoContent(), nil
}

func checkReportAccess(reportID ReportID) error {
	switch reportID {
	case "monthly":
		return nil
	case "failure":
		return errorx.WithStack(errors.New("load report access: storage unavailable"))
	case "secret":
		return ohmyhttp.NewError(
			http.StatusForbidden,
			"report is forbidden",
			ohmyhttp.WithCode("permission_denied"),
		)
	default:
		return newReportNotFound(string(reportID))
	}
}

func sampleReportPath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("examples", "demo", "files", "monthly-report.txt")
	}
	return filepath.Join(filepath.Dir(file), "files", "monthly-report.txt")
}
