package codec

import (
	"MRPC/codec/codec"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

const MagicNumber = 0x3bef5c

// 为了实现上更简单，客户端固定采用 JSON 编码 Option，
// 后续的 header 和 body 的编码方式由 Option 中的 CodeType 指定，
type Option struct {
	MagicNumber int
	CodecType   codec.Type //消息的编解码方式
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

// 临时
var DefaultServer = NewServer()

func (server *Server) Accept(lis net.Listener) {
	//for 循环等待socket连接建立，并开启子协程处理
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server:accept error:", err)
			return
		}
		go server.ServeConn(conn)
	}
}

func Accept(lis net.Listener) { DefaultServer.Accept(lis) }

// 连接
func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	defer func() { _ = conn.Close() }()
	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: option error:", err)
		return
	}
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server:invalid magic number %x", opt.MagicNumber)
		return
	}
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Printf("rpc server:invalid codec type %s", opt.CodecType)
		return
	}
	server.serveCodec(f(conn))
}

var invalidRequest = struct{}{}

func (server *Server) serveCodec(cc codec.Codec) {
	//并发处理请求，需要上锁
	sending := new(sync.Mutex)
	wg := new(sync.WaitGroup)
	for {
		//读取请求
		//没有
		req, err := server.readRequest(cc)
		if err != nil {
			if req == nil {
				break
			}
			//body读取失败
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		go server.handleRequest(cc, req, sending, wg)
	}
	wg.Wait()
	_ = cc.Close()
}

type request struct {
	h *codec.Header
	//用反射，动态调用服务端注册的方法
	argv, replyv reflect.Value
}

func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}

func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: h}
	//TODO:目前还不能判断body的类型，暂时将body作为字符串处理
	req.argv = reflect.New(reflect.TypeOf(""))
	if err = cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc serve:read argv error:", err)
	}
	return req, nil
}

func (server *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server:write response error:", err)
	}
}

func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	//TODO:这里以后会调用真正注册的RPC方法，得到真正的返回值
	// 现在这里只打印参数，返回一个Hello
	//等待所有请求处理完，通知外面WaitGroup
	defer wg.Done()
	log.Println(req.h, req.argv.Elem()) //如果是指针那就拿到指针指向的值
	req.replyv = reflect.ValueOf(fmt.Sprintf("MRPC resp %d", req.h.Seq))
	server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
}
