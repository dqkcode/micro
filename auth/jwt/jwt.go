// Package jwt implements authentication interfaces using JWT.
package jwt

import (
	"context"
	"errors"
	"strings"

	"github.com/pthethanh/micro/auth"

	"github.com/dgrijalva/jwt-go"
	"google.golang.org/grpc/metadata"
)

var (
	// errors
	errMetadataMissing      = errors.New("jwt: could not locate request metadata")
	errAuthorizationMissing = errors.New("jwt: could not locate authorization metadata")
	errMultipleAuthFound    = errors.New("jwt: too many authorization entries")
	errInvalidToken         = errors.New("jwt: invalid token")

	// Lookup key for authorization metadata
	authorizationMd = "authorization"
)

// Claims represents the claims provided by the JWT.
type Claims struct {
	Scope     string `json:"scope,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	ClientID  string `json:"client_id,omitempty"`
	TokenType string `json:"token_type,omitempty"`
	AvatarURL string `json:"avatar_url"`
	FullName  string `json:"full_name"`

	// Once we have service-accounts in place, this should be removed.
	// Its up to each service to decide how they would like to handle
	// admin-callers.
	Admin bool `json:"admin,omitempty"`
	jwt.StandardClaims
}

// ContainScopes checks if `scopes` are present within the Claim.Scope.
func (c Claims) ContainScopes(scopes ...string) bool {
	currentScopes := strings.Split(c.Scope, " ")
	if len(currentScopes) == 0 {
		return false
	}
	for _, scope := range scopes {
		match := false
		for _, s := range currentScopes {
			if scope == s {
				match = true
			}
		}
		if !match {
			return false
		}
	}
	return true
}

// Authenticator returns an AuthenticatorFunc that
// validates the provided JWT token in the :authorization header
// of the metadata.
func Authenticator(secret []byte) auth.AuthenticatorFunc {
	return func(ctx context.Context) (context.Context, error) {
		var claims Claims
		var newCtx context.Context
		if err := ParseFromMetadata(ctx, secret, &claims); err != nil {
			return newCtx, err
		}
		newCtx = NewContext(ctx, claims)
		return newCtx, nil
	}
}

// ParseFromMetadata fetches the JWT from the :authorization metadata located
// in the `Context`, validates the JWT and extracts the Claims.
func ParseFromMetadata(ctx context.Context, secret []byte, c jwt.Claims) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return errMetadataMissing
	}
	slice, ok := md[authorizationMd]
	if !ok || len(slice) == 0 {
		return errAuthorizationMissing
	}
	if len(slice) > 1 {
		return errMultipleAuthFound
	}
	return Parse(slice[0], secret, c)
}

// Parse and validate a JWT string.
func Parse(t string, s []byte, c jwt.Claims) error {
	_, err := jwt.ParseWithClaims(t, c, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errInvalidToken
		}
		return s, nil
	})
	if err != nil {
		return errInvalidToken
	}
	return c.Valid()
}

// Encode encodes the jwt Claim to a JWT string.
func Encode(c jwt.Claims, secret []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(secret)
}

// The context key
type claimsKey struct{}

// NewContext creates a new context with the claims attached.
func NewContext(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, claimsKey{}, claims)
}

// FromContext fetches the claims attched to the ctx.
func FromContext(ctx context.Context) (c Claims, ok bool) {
	c, ok = ctx.Value(claimsKey{}).(Claims)
	return
}

// SubjectEquals checks if the JWT subject is equal to the provided
// subject in `sub`.
func SubjectEquals(ctx context.Context, s string) bool {
	if t, ok := FromContext(ctx); ok {
		return t.Subject == s
	}
	return false
}

// TokenString extracts the JWT toke as a string from `ctx`.
func TokenString(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	slice, ok := md[authorizationMd]
	if !ok || len(slice) == 0 {
		return ""
	}
	if len(slice) > 1 {
		return ""
	}
	return slice[0]
}
