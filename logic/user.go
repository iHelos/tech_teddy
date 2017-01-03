package logic

import (
	"github.com/iHelos/tech_teddy/model"
	"strings"
	"github.com/kataras/iris"
	"github.com/dgrijalva/jwt-go"
	"log"
	"github.com/tarantool/go-tarantool"
	. "github.com/iHelos/tech_teddy/helper"
	"golang.org/x/crypto/bcrypt"
	"fmt"
	"github.com/kataras/go-errors"
)

const (
	hmacUserSecret = "95CCEB5921E59B285AC773E4963E1"
)
func ParseToken(ctx *iris.Context) (int, error){
	signedtoken := ctx.Request.Header.Get("Authorization")
	log.Print(signedtoken)
	token, err := jwt.Parse(string(signedtoken), func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return "", fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(hmacUserSecret), nil
	})
	if (err!=nil){
		return 0, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if login, ok := claims["id"].(float64); ok {
			return int(login), nil
		} else {
			return 0, errors.New("bad jwt")
		}
	} else {
		return 0,err
	}
}

func Register(ctx *iris.Context) (string, string, error){
	var user = model.NewProfile{}
	var err error
	err = ctx.ReadJSON(&user)
	if err != nil {
		log.Print(err)
		UserError := NewError()
		UserError.Append("request", 0)
		return "","",UserError
	}
	user.Email = strings.ToLower(user.Email)
	err = user.Validate()
	if err != nil {
		return "","",err
	}
	hashpassword, _ := bcrypt.GenerateFromPassword([]byte(user.Password1), bcrypt.DefaultCost)
	var profile = model.Profile{Email:user.Email, Name:user.Login, Password:string(hashpassword)}
	created, err := model.CreateProfile(profile)
	if err != nil {
		UserError := NewError()
		if trnerror, ok := err.(tarantool.Error); ok {
			if trnerror.Code == 32771 || trnerror.Code == 3 {
				UserError.Append("email", 1)
				return "","", UserError
			}
		}
		UserError.Append("DB", 0)
		return "","", UserError
	}
	//ctx.Session().Set("name", user.Login)
	userToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": created.ID,
		"type":"user",
	})
	bearToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": created.ID,
		"type":"bear",
	})
	userTokenSigned, err := userToken.SignedString([]byte(hmacUserSecret))
	bearTokenSigned, err := bearToken.SignedString([]byte(hmacUserSecret))

	return userTokenSigned,bearTokenSigned,nil
}
func Login(ctx *iris.Context) (string, string, error){
	var user model.Profile
	err := ctx.ReadJSON(&user)
	if err != nil {
		UserError := NewError()
		UserError.Append("request", 0)
		return "","", UserError
	}
	user.Email = strings.ToLower(user.Email)
	id, err := model.CheckPassword(user)
	if err != nil {
		log.Print(err)
		UserError := NewError()
		UserError.Append("request", 1)
		return "","",UserError
	}
	userToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": id,
		"type":"user",
	})
	bearToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": id,
		"type":"bear",
	})
	userTokenSigned, err := userToken.SignedString([]byte(hmacUserSecret))
	bearTokenSigned, err := bearToken.SignedString([]byte(hmacUserSecret))
	return userTokenSigned, bearTokenSigned, nil
}

type StoryPointer struct {
	StoryID int `json:"storyID"`
}
func LikeStory(ctx *iris.Context)(error){
	id, err := ParseToken(ctx)
	if err != nil{
		return err
	}
	story := StoryPointer{}
	err = ctx.ReadJSON(&story)
	if err != nil{
		return err
	}
	err = model.AddStory(id, story.StoryID)
	return err
}

