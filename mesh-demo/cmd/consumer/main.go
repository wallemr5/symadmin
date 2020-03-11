package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/apache/dubbo-go/common/logger"

	_ "github.com/apache/dubbo-go/common/proxy/proxy_factory"
	"github.com/apache/dubbo-go/config"

	_ "github.com/apache/dubbo-go/protocol/dubbo"

	_ "github.com/apache/dubbo-go/registry/protocol"

	_ "github.com/apache/dubbo-go/filter/filter_impl"

	_ "github.com/apache/dubbo-go/cluster/cluster_impl"

	_ "github.com/apache/dubbo-go/cluster/loadbalance"

	"context"

	hessian "github.com/apache/dubbo-go-hessian2"
	_ "github.com/apache/dubbo-go/registry/zookeeper"
	"gitlab.dmall.com/arch/sym-admin/mesh-demo/pkg/consumer"
	"gitlab.dmall.com/arch/sym-admin/mesh-demo/pkg/model"
)

// they are necessary:
// 		export CONF_CONSUMER_FILE_PATH="xxx"
// 		export APP_LOG_CONF_FILE="xxx"
func main() {
	hessian.RegisterPOJO(&model.User{})
	config.Load()
	r := consumer.GinInit()

	ctx, cancelFunc := context.WithCancel(context.Background())
	go r.StartWarp(ctx.Done())
	initSignal(cancelFunc)
}

func initSignal(cancelFunc func()) {
	signals := make(chan os.Signal, 1)
	// It is not possible to block SIGKILL or syscall.SIGSTOP
	signal.Notify(signals, os.Interrupt, os.Kill, syscall.SIGHUP,
		syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		sig := <-signals
		logger.Infof("get signal %s", sig.String())
		switch sig {
		case syscall.SIGHUP:
			// reload()
		default:
			logger.Info("call cancelFunc")
			cancelFunc()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			<-ctx.Done()
			cancel()
			// The program exits normally or timeout forcibly exits.
			logger.Info("app exit now...")
			return
		}
	}
}
