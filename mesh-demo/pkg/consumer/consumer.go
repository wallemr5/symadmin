package consumer

import (
	"context"

	"net/http"

	"time"

	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/mesh-demo/pkg/model"
	"gitlab.dmall.com/arch/sym-admin/mesh-demo/pkg/router"
)

var UserClient = new(UserConsumer)

func init() {
	config.SetConsumerService(UserClient)
	hessian.RegisterPOJO(&model.User{})
}

type UserConsumer struct {
	GetUser func(ctx context.Context, req []interface{}, rsp *model.User) error
}

func (u *UserConsumer) Reference() string {
	return "UserProvider"
}

func (u *UserConsumer) GetUserDubbo(c *gin.Context) {
	logger.Debugf("start to test dubbo get user")
	user := &model.User{}
	err := u.GetUser(context.TODO(), []interface{}{"A001"}, user)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   err.Error(),
			"resultMap": nil,
		})
		return
	}
	logger.Debugf("dubbo response result: %#v\n", user)
	c.IndentedJSON(http.StatusOK, gin.H{
		"success":   true,
		"resultMap": user,
	})
}

func (u *UserConsumer) GetUserHttp(c *gin.Context) {
	logger.Debugf("start to test http get user")

	c.IndentedJSON(http.StatusOK, gin.H{
		"success": true,
		"Id":      "A001",
		"Name":    "Alex Stocks",
		"Age":     18,
		"Time":    time.Now(),
	})
}

// LiveHandler ...
func LiveHandler(c *gin.Context) {
	c.String(http.StatusOK, "ok")
}

func (u *UserConsumer) Routes() []*router.Route {
	var routes []*router.Route

	ctlRoutes := []*router.Route{
		{
			Method:  "GET",
			Path:    "/userhttp",
			Handler: u.GetUserHttp,
		},
		{
			Method:  "GET",
			Path:    "/userdubbo",
			Handler: u.GetUserDubbo,
		},
		{
			Method:  "GET",
			Path:    "/live",
			Handler: LiveHandler,
		},
		{
			Method:  "GET",
			Path:    "/ready",
			Handler: LiveHandler,
		},
	}

	routes = append(routes, ctlRoutes...)
	return routes
}

//
func GinInit() *router.Router {
	rt := router.NewRouter(router.DefaultOption())
	rt.AddRoutes("index", rt.DefaultRoutes())
	rt.AddRoutes("user", UserClient.Routes())
	return rt
}
