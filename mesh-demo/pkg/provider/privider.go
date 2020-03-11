package provider

import (
	"context"
	"log"
	"time"

	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/apache/dubbo-go/config"
	"gitlab.dmall.com/arch/sym-admin/mesh-demo/pkg/model"
	"istio.io/pkg/env"
)

func init() {
	config.SetProviderService(new(UserProvider))
	// ------for hessian2------
	hessian.RegisterPOJO(&model.User{})
}

var (
	PodName      = env.RegisterStringVar("POD_NAME", "dubbo-test-xxx", "pod name")
	PodNamespace = env.RegisterStringVar("POD_NAMESPACE", "default", "pod namespace")
	PodIp        = env.RegisterStringVar("POD_IP", "0.0.0.0", "pod ip")
)

type UserProvider struct {
}

func (u *UserProvider) GetUser(ctx context.Context, req []interface{}) (*model.User, error) {
	log.Printf("req:%#v", req)
	rsp := model.User{
		Id:           "v2",
		Name:         "Alex Stocks",
		Age:          18,
		PodName:      PodName.Get(),
		PodNamespace: PodNamespace.Get(),
		PodIp:        PodIp.Get(),
		Time:         time.Now()}
	log.Printf("rsp:%#v", rsp)
	return &rsp, nil
}

func (u *UserProvider) Reference() string {
	return "UserProvider"
}
