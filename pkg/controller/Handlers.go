package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	request "github.com/Onelvay/docker-compose-project/payment/APIrequest"
	client "github.com/Onelvay/docker-compose-project/payment/client"
	"github.com/Onelvay/docker-compose-project/pkg/domain"
	service "github.com/Onelvay/docker-compose-project/pkg/service"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type Transactioner interface {
	CreateOrder(userId string, orderId string)
	// CreateInfoOrder(string, string)
	CreateInfoOrder(request.FinalResponse)
}

type HandleFunctions struct {
	db             service.BookstorePostgreser
	userController service.UserController
	order          Transactioner
}

func NewHandlers(db service.BookstorePostgreser, userController service.UserController, or Transactioner) *HandleFunctions {
	return &HandleFunctions{db, userController, or}
}

type Transaction struct {
	Product_id string
	User_id    string //временно
}

func (s *HandleFunctions) CreateOrder(w http.ResponseWriter, r *http.Request) {
	reqBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	var inp Transaction
	if err = json.Unmarshal(reqBytes, &inp); err != nil {
		panic(err)
	}
	product, _ := s.db.GetBookById(inp.Product_id)
	byteid := uuid.New()
	id := strings.Replace(byteid.String(), "-", "", -1)
	price := fmt.Sprintf("%v", product.Price)
	checkoutRequest := &request.CheckoutRequest{
		OrderId:           id,
		MerchantId:        "1396424",
		OrderDesc:         product.Description,
		Amount:            price,
		ProductId:         product.Id,
		Currency:          "USD",
		ServerCallbackURL: "https://8d8d-80-242-211-178.in.ngrok.io/callback",
	}
	api := client.CreateOrder(*checkoutRequest)
	s.order.CreateOrder(inp.User_id, id)
	json.NewEncoder(w).Encode(api)

}

func (s *HandleFunctions) Callback(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)

	apiResp := request.FinalResponse{}
	json.Unmarshal(body, &apiResp)
	s.order.CreateInfoOrder(apiResp)
}
func (s *HandleFunctions) GetBooks(w http.ResponseWriter, r *http.Request) {
	URLsort := r.URL.Query().Get("sorted")
	sort := false
	if URLsort == "true" {
		sort = true
	}
	books := s.db.GetBooks(sort)
	json.NewEncoder(w).Encode(books)
}
func (s *HandleFunctions) GetBookById(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	book, _ := s.db.GetBookById(id)

	json.NewEncoder(w).Encode(book)
}
func (s *HandleFunctions) GetBooksByName(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	books, _ := s.db.GetBooksByName(name)
	json.NewEncoder(w).Encode(books)
}
func (s *HandleFunctions) DeleteBookById(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	res := s.db.DeleteBookById(id)
	if res {
		fmt.Fprintf(w, "успешно")
	} else {
		fmt.Fprintf(w, "не успешно")
	}
}
func (s *HandleFunctions) CreateBook(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	desc := r.URL.Query().Get("desc")
	price_str := r.URL.Query().Get("price")
	price, _ := strconv.ParseFloat(price_str, 64)
	if name != "" && desc != "" && price != 0 && s.db.CreateBook(name, price, desc) {
		fmt.Fprintf(w, "успешно")
	} else {
		fmt.Fprintf(w, "не успешно")
	}
}

func (s *HandleFunctions) UpdateBook(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	name := r.URL.Query().Get("name")
	desc := r.URL.Query().Get("desc")
	price_str := r.URL.Query().Get("price")
	price, _ := strconv.ParseFloat(price_str, 64)
	res := s.db.UpdateBook(id, name, desc, price)
	if res {
		fmt.Fprintf(w, "успешно")
	} else {
		fmt.Fprintf(w, "не успешно")
	}

}
func (s *HandleFunctions) SignUp(w http.ResponseWriter, r *http.Request) {
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
func (h *HandleFunctions) Refresh(w http.ResponseWriter, r *http.Request) {
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
func (s *HandleFunctions) SignIn(w http.ResponseWriter, r *http.Request) {
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

func (s *HandleFunctions) AuthMiddleware(next http.Handler) http.Handler {
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
