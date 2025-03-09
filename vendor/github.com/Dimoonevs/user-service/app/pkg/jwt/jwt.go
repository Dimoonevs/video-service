package jwt

import (
	"errors"
	"flag"
	"fmt"
	"github.com/Dimoonevs/video-service/app/pkg/respJSON"
	"github.com/golang-jwt/jwt/v4"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"net/http"
	"strings"
	"time"
)

var (
	secretKeyFlag = flag.String("secretKey", "", "secret key")
)

func GenerateJWT(email string, id int) (string, error) {
	claims := jwt.MapClaims{
		"userID": id,
		"email":  email,
		"exp":    time.Now().Add(time.Hour * 24 * 30).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	if secretKeyFlag == nil && *secretKeyFlag == "" {
		logrus.Errorf("secretKeyFlag is nil")
		return "", errors.New("secret key is required")
	}
	return token.SignedString([]byte(*secretKeyFlag))
}

func JWTMiddleware(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		authHeader := string(ctx.Request.Header.Peek("Authorization"))
		if authHeader == "" {
			respJSON.WriteJSONError(ctx, http.StatusUnauthorized, fmt.Errorf("Unauthorized: "), "Missing token")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			respJSON.WriteJSONError(ctx, http.StatusUnauthorized, fmt.Errorf("Unauthorized: "), "Invalid token")
			return
		}

		tokenStr := parts[1]

		claims, err := parseJWT(tokenStr)
		if err != nil {
			respJSON.WriteJSONError(ctx, http.StatusUnauthorized, err, "\"error\":")
			return
		}

		ctx.SetUserValue("userID", claims["userID"])
		ctx.SetUserValue("email", claims["email"])

		next(ctx)
	}
}

func parseJWT(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(*secretKeyFlag), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	expiration, ok := claims["exp"].(float64)
	if !ok || int64(expiration) < time.Now().Unix() {
		return nil, errors.New("token expired")
	}

	return claims, nil
}
