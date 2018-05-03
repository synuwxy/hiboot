package web

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/hidevopsio/hiboot/pkg/log"
	"net/http"
	"github.com/hidevopsio/hiboot/pkg/starter/web/jwt"
	"time"
)

type UserRequest struct {
	Username string
	Password string
}

type FooRequest struct {
	Name string
}

type FooResponse struct {
	Greeting string
}

type Bar struct {
	Name string
	Greeting string
}

type FooController struct{
	Controller
}

type BarController struct{
	Controller
}

func (c *FooController) PostLogin(ctx *Context)  {
	log.Debug("FooController.SayHello")

	userRequest := &UserRequest{}
	if ctx.RequestBody(userRequest) == nil {
		jwtToken, err := ctx.GenerateJwtToken(jwt.Map{
			"username": userRequest.Username,
			"password": userRequest.Password,
		}, 10, time.Minute)

		log.Debugf("token: %v", jwtToken)

		if err == nil {
			ctx.Response("Success", jwtToken)
		} else {
			ctx.ResponseError(err.Error(), http.StatusInternalServerError)
		}
	}
}

func (c *FooController) PostSayHello(ctx *Context)  {
	log.Debug("FooController.SayHello")

	foo := &FooRequest{}
	if ctx.RequestBody(foo) == nil {
		ctx.Response("Success", &FooResponse{Greeting: "hello, " + foo.Name})
	}
}

func (c *BarController) GetSayHello(ctx *Context)  {
	log.Debug("BarController.SayHello")

	ctx.Response("Success", &Bar{Greeting: "hello bar"})

}

type Controllers struct{

	Foo *FooController `auth:"anon"`
	Bar *BarController `controller:"bar" auth:"anon"`
}

func TestWebApplication(t *testing.T)  {

	controllers := &Controllers{}
	wa, err := NewApplication(controllers)
	assert.Equal(t, nil, err)

	e := wa.NewTestServer(t)

	e.Request("POST", "/foo/login").WithJSON(&UserRequest{Username: "johndoe", Password: "iHop91#15"}).
		Expect().Status(http.StatusOK).Body().Contains("Success")

	e.Request("POST", "/foo/sayHello").WithJSON(&FooRequest{Name: "John"}).
		Expect().Status(http.StatusOK).Body().Contains("Success")

	e.Request("GET", "/bar/sayHello").
		Expect().Status(http.StatusOK).Body().Contains("Success")
}
