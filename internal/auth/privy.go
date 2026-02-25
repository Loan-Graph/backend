package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type PrivyVerifier interface {
	VerifyAccessToken(ctx context.Context, accessToken string) (*Identity, error)
}

type PrivyTokenVerifier struct {
	issuer          string
	audience        string
	verificationKey string
	jwksURL         string
	httpClient      *http.Client
}

type privyClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	WalletAddress string `json:"wallet_address"`
	jwt.RegisteredClaims
}

type jwks struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func NewPrivyTokenVerifier(issuer, audience, verificationKey, jwksURL string) *PrivyTokenVerifier {
	return &PrivyTokenVerifier{
		issuer:          issuer,
		audience:        audience,
		verificationKey: verificationKey,
		jwksURL:         jwksURL,
		httpClient:      &http.Client{Timeout: 5 * time.Second},
	}
}

func (v *PrivyTokenVerifier) VerifyAccessToken(ctx context.Context, accessToken string) (*Identity, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("missing access token")
	}

	claims := &privyClaims{}
	token, err := jwt.ParseWithClaims(accessToken, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("unexpected signing method")
		}

		if strings.TrimSpace(v.verificationKey) != "" {
			return jwt.ParseRSAPublicKeyFromPEM([]byte(v.verificationKey))
		}
		if strings.TrimSpace(v.jwksURL) == "" {
			return nil, errors.New("no privy verification key configured")
		}
		return v.keyFromJWKS(ctx, token)
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid privy token: %w", err)
	}

	if strings.TrimSpace(claims.Subject) == "" {
		return nil, errors.New("missing subject claim")
	}
	if strings.TrimSpace(v.issuer) != "" && claims.Issuer != v.issuer {
		return nil, errors.New("invalid issuer")
	}
	if strings.TrimSpace(v.audience) != "" {
		ok := false
		for _, aud := range claims.Audience {
			if aud == v.audience {
				ok = true
				break
			}
		}
		if !ok {
			return nil, errors.New("invalid audience")
		}
	}

	return &Identity{
		Subject:       claims.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		WalletAddress: claims.WalletAddress,
	}, nil
}

func (v *PrivyTokenVerifier) keyFromJWKS(ctx context.Context, token *jwt.Token) (*rsa.PublicKey, error) {
	kidRaw, ok := token.Header["kid"]
	if !ok {
		return nil, errors.New("missing kid")
	}
	kid, ok := kidRaw.(string)
	if !ok || kid == "" {
		return nil, errors.New("invalid kid")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("jwks fetch failed: %d", resp.StatusCode)
	}

	var set jwks
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return nil, err
	}

	for _, key := range set.Keys {
		if key.Kid != kid || key.Kty != "RSA" {
			continue
		}
		return buildRSAPublicKey(key.N, key.E)
	}
	return nil, errors.New("signing key not found")
}

func buildRSAPublicKey(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, err
	}

	n := new(big.Int).SetBytes(nBytes)
	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}
	if e == 0 {
		return nil, errors.New("invalid exponent")
	}

	return &rsa.PublicKey{N: n, E: e}, nil
}
