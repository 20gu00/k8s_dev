package mycsi

import (
	"net"
	"net/url"
	"os"
	"sync"

	"github.com/golang/glog"
	"google.golang.org/grpc"

	csi "github.com/container-storage-interface/spec/lib/go/csi"
)

// 非阻塞的 server 的 interface
// server 本身并不阻塞,不像 gin 的 http server
type NonBlockingGRPCServer interface {
	// 启动服务在endpoint,endpoint处提供服务 暴露服务
	Start(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer)
	// 阻塞
	Wait()
	// 优雅关闭
	Stop()
	// 强制关闭
	ForceStop()
}

// 实例化一个 grpc server interface
func NewNonBlockingGRPCServer() NonBlockingGRPCServer {
	return &nonBlockingGRPCServer{} // 结构体(指针) 该结构实现接口
}

// 非阻塞的 server 对象结构体定义
type nonBlockingGRPCServer struct {
	wg     sync.WaitGroup
	server *grpc.Server
}

func (s *nonBlockingGRPCServer) Start(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer) {
	s.wg.Add(1)

	go s.serve(endpoint, ids, cs, ns)

	return
}

func (s *nonBlockingGRPCServer) Wait() {
	s.wg.Wait()
}

func (s *nonBlockingGRPCServer) Stop() {
	// grpc server 优雅关闭
	s.server.GracefulStop()
}

func (s *nonBlockingGRPCServer) ForceStop() {
	s.server.Stop()
}

// serve 提供服务, gin 等等的server也是这么设计
func (s *nonBlockingGRPCServer) serve(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer) {
	// 配置选项
	opts := []grpc.ServerOption{
		// 统一拦截器
		grpc.UnaryInterceptor(logGRPC),
	}
	// 解析成 url 结构,有无主机名都可以,主要是路径解析
	// 注意解析不明确,不一定返回错误
	u, err := url.Parse(endpoint)
	if err != nil {
		glog.Fatal(err.Error())
	}

	var addr string
	// unix 模式(socket)
	if u.Scheme == "unix" {
		addr = u.Path
		// 删除命名文件或者空目录,错误类型是*PathError
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			glog.Fatalf("删除 %s 失败, error: %s", addr, err.Error())
		}
		// tcp 即可
	} else if u.Scheme == "tcp" {
		addr = u.Host
	} else {
		glog.Fatalf("%v endpoint 模式不支持", u.Scheme)
	}

	glog.V(4).Infof("开始listening scheme:%v, addr:%v", u.Scheme, addr)
	// 服务监听 listen 监听这个地址
	listener, err := net.Listen(u.Scheme, addr)
	if err != nil {
		glog.Fatalf("建立 listen 失败: %v", err)
	}

	// 变长参数
	// 实例化一个 grpc server
	server := grpc.NewServer(opts...)
	s.server = server

	// 设置 csi 存储插件所需的 csi-identity csi-controller csi-node
	if ids != nil {
		csi.RegisterIdentityServer(server, ids)
	}
	if cs != nil {
		csi.RegisterControllerServer(server, cs)
	}
	if ns != nil {
		csi.RegisterNodeServer(server, ns)
	}

	glog.V(4).Infof("正在监听这个地址连接提供的服务: %#v", listener.Addr())
	// 根据 listen 的地址 serve  (listen-->serve,根据地址做个listener,根据listenr做serve)
	if err := server.Serve(listener); err != nil {
		glog.Fatalf("服务 serve 失败: %v", err)
	}
}
