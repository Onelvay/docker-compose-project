package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/Onelvay/docker-compose-project/pkg/domain"
	"github.com/Onelvay/docker-compose-project/pkg/service"
	"github.com/sirupsen/logrus"
)

type UserHandler struct {
	userController service.UserController
}

func NewUserHandler(userController service.UserController) UserHandler {
	return UserHandler{userController}
}
func (s *UserHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	reqBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	var inp domain.SignUpInput
	if err = json.Unmarshal(reqBytes, &inp); err != nil {
		panic(err)
	}
	if err := inp.Validate(); err != nil {
		panic(err)
	}
	s.userController.SignUp(r.Context(), inp)
	w.WriteHeader(http.StatusOK)

}
func (h *UserHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh-token")
	if err != nil {
		panic(err)
	}
	logrus.Infof("%s", cookie.Value)

	accessToken, refreshToken, err := h.userController.RefreshTokens(r.Context(), cookie.Value)
	if err != nil {
		panic(err)
	}
	responce, err := json.Marshal(map[string]string{
		"token": accessToken,
	})
	if err != nil {
		panic(err)
	}
	w.Header().Add("Set-Cookie", fmt.Sprintf("refresh-token=%s; HttpOnly", refreshToken))
	w.Header().Add("Content-Type", "application/json")
	w.Write(responce)

}
func (s *UserHandler) SignIn(w http.ResponseWriter, r *http.Request) {
	reqBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	var inp domain.SignInInput
	if err = json.Unmarshal(reqBytes, &inp); err != nil {
		panic(err)
	}
	if err := inp.Validate(); err != nil {
		panic(err)
	}
	accessToken, refreshToken, err := s.userController.SignIn(r.Context(), inp)
	if err != nil {
		panic(err)
	}
	responce, err := json.Marshal(map[string]string{
		"token": accessToken,
	})
	if err != nil {
		panic(err)
	}
	w.Header().Add("Set-Cookie", fmt.Sprintf("refresh-token=%s; HttpOnly", refreshToken))
	w.Header().Add("Content-Type", "application/json")
	w.Write(responce)
}

type key int

func (s *UserHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := getTokenFromRequest(r)
		if err != nil {
			panic(err)
		}
		userId, err := s.userController.ParseToken(r.Context(), token)
		if err != nil {
			panic(err)
		}

		var ctxUserId key
		ctx := context.WithValue(r.Context(), ctxUserId, userId)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}
func getTokenFromRequest(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", errors.New("empty auth header")
	}
	headerParts := strings.Split(header, " ")
	return headerParts[1], nil
}
