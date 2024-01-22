package main

import (
	"context"
	"fmt"
	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/gin-gonic/gin"
	"github.com/sjmshsh/xds/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"log"
	"net"

	"os"
	"time"
)

func main() {
	var grpcOptions []grpc.ServerOption
	grpcOptions = append(grpcOptions,
		grpc.MaxConcurrentStreams(1000), //一条GRPC连接允许并发的发送和接收多个Stream
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    time.Second * 30, //连接超过多少时间 不活跃，则会去探测 是否依然alive
			Timeout: time.Second * 5,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             time.Second * 30, //发送ping之前最少要等待 时间
			PermitWithoutStream: true,             //连接空闲时仍然发送PING帧监测
		}),
	)
	//创建grpc 服务
	grpcServer := grpc.NewServer(grpcOptions...)

	//日志
	llog := utils.MyLogger{}
	// 创建缓存系统
	c := cache.NewSnapshotCache(false, cache.IDHash{}, llog)
	// envoy配置的缓存快照, 1是版本号, 通过版本号的变更进行配置更新
	// 该函数后续会有更完整的文件内容
	snapshot := utils.GenerateSnapshot("1")
	if err := snapshot.Consistent(); err != nil {
		llog.Errorf("snapshot inconsistency: %+v\n%+v", snapshot, err)
		os.Exit(1)
	}

	// Add the snapshot to the cache
	// nodeID必须要设置
	nodeID := "test1"
	if err := c.SetSnapshot(context.Background(), nodeID, snapshot); err != nil {
		os.Exit(1)
	}

	// 请求回调, 类似于中间件
	cb := utils.Callbacks{Debug: llog.Debug}

	// 官方提供的控制面server
	srv := server.NewServer(context.Background(), c, &cb)
	// 注册 集群服务
	clusterservice.RegisterClusterDiscoveryServiceServer(grpcServer, srv)
	// 注册 listener
	listenerservice.RegisterListenerDiscoveryServiceServer(grpcServer, srv)
	// 由于在listener下的需要创建路由, 所以需要加入
	routeservice.RegisterRouteDiscoveryServiceServer(grpcServer, srv)

	errCh := make(chan error)

	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 9090))
		if err != nil {
			errCh <- err
			return
		}
		if err = grpcServer.Serve(lis); err != nil {
			errCh <- err
		}
	}()
	// 启动动态测试服务, 你可以通过请求 /test 进行版本更替
	go func() {
		r := gin.New()
		r.GET("/test", func(ctx *gin.Context) {
			// 如果你部署 2个 nginx 容器, 可以通过这个 IP 的调整测试出是否成功从代理 v1 nginx 转成代理  v2 nginx
			utils.UpstreamHost = "172.17.0.7"
			// 通过版本控制snapshot 的更新
			ss := utils.GenerateSnapshot("2")
			if err := c.SetSnapshot(ctx, nodeID, ss); err != nil {
				ctx.String(400, err.Error())
				return
			}
			ctx.String(200, "OK")

		})

		if err := r.Run("0.0.0.0:18000"); err != nil {
			errCh <- err
		}

	}()

	err := <-errCh
	log.Fatal(err)
}
