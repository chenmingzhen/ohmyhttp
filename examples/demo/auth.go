package demo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chenmingzhen/ohmyhttp"
	"github.com/golang-jwt/jwt/v5"
)

const (
	demoJWTIssuer = "ohmyhttp-demo"
	demoJWTSecret = "ohmyhttp-demo-secret"
)

const (
	refreshAccessTokenHeader     = "Refresh-Access-Token"
	refreshAccessExpiresAtHeader = "Refresh-Access-Expires-At"
)

type demoClaims struct {
	User User `json:"user"`
	jwt.RegisteredClaims
}

type JWTConfig struct {
	Secret        []byte
	ExcludeRoutes []string
	Authorize     func(context.Context, User) error
	RefreshBefore time.Duration
	RefreshExpire time.Duration
}

func JWTAuthentication(config JWTConfig) ohmyhttp.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if matchesAnyRoute(config.ExcludeRoutes, request.URL.Path) {
				next.ServeHTTP(writer, request)
				return
			}

			token := authToken(request)
			if token == "" {
				writeAuthorizationError(
					writer,
					request,
					errors.New("authorization token is required"),
				)
				return
			}

			claims, err := parseDemoToken(config.Secret, token)
			if err != nil {
				writeAuthorizationError(writer, request, err)
				return
			}
			if config.Authorize != nil {
				if err := config.Authorize(request.Context(), claims.User); err != nil {
					writeAuthorizationError(writer, request, err)
					return
				}
			}

			ctx := context.WithValue(request.Context(), userKey{}, claims.User)
			refreshTokenIfNeeded(writer, config, claims)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

func issueDemoToken(user User, expiresIn time.Duration) (string, error) {
	return signDemoToken([]byte(demoJWTSecret), user, expiresIn)
}

func signDemoToken(secret []byte, user User, expiresIn time.Duration) (string, error) {
	now := time.Now()
	claims := demoClaims{
		User: user,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    demoJWTIssuer,
			Subject:   user.Name,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

func parseDemoToken(secret []byte, value string) (*demoClaims, error) {
	claims := new(demoClaims)
	_, err := jwt.ParseWithClaims(
		value,
		claims,
		func(*jwt.Token) (any, error) { return secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(demoJWTIssuer),
	)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func authToken(request *http.Request) string {
	value := request.Header.Get("Authorization")
	if scheme, token, ok := strings.Cut(value, " "); ok && strings.EqualFold(scheme, "Bearer") {
		return strings.TrimSpace(token)
	}
	return request.URL.Query().Get("token")
}

func refreshTokenIfNeeded(
	writer http.ResponseWriter,
	config JWTConfig,
	claims *demoClaims,
) {
	if claims.ExpiresAt == nil || config.RefreshBefore <= 0 || config.RefreshExpire <= 0 {
		return
	}
	if time.Until(claims.ExpiresAt.Time) >= config.RefreshBefore {
		return
	}

	token, err := signDemoToken(config.Secret, claims.User, config.RefreshExpire)
	if err != nil {
		return
	}
	writer.Header().Set(refreshAccessTokenHeader, token)
	writer.Header().Set(
		refreshAccessExpiresAtHeader,
		strconv.FormatInt(int64(config.RefreshExpire/time.Second), 10),
	)
}

func writeAuthorizationError(
	writer http.ResponseWriter,
	request *http.Request,
	err error,
) {
	problem := ohmyhttp.Problem{
		Code:    "unauthorized",
		Message: "authorization failed, please log in",
	}
	setAccessResponse(request.Context(), problem, err)
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(writer).Encode(problem)
}
