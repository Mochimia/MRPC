package codec

import (
	"io"
)

type Header struct {
	ServiceMethod string //服务名和方法
	Seq           uint64 //请求的序号
	Error         string
}

// 对消息体进行编解码的接口，实现不同的Codec 实例
type Codec interface {
	io.Closer
	ReadHeader(*Header) error         //读取一条RPC 消息的头部
	ReadBody(interface{}) error       //读取RPC消息体
	Write(*Header, interface{}) error //编码
}

type NewCodecFunc func(io.ReadWriteCloser) Codec

type Type string

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json" //没实现
)

var NewCodecFuncMap map[Type]NewCodecFunc

func init() {
	//准备一个空的注册表
	NewCodecFuncMap = make(map[Type]NewCodecFunc) //map[Type]NewCodecFunc初始为nil，不能直接赋值
	//往注册表里放入 Gob 编解码器的构造函数
	NewCodecFuncMap[GobType] = NewGobCodec
}
