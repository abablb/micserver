package connect

import (
	"github.com/liasece/micserver/log"
	"github.com/liasece/micserver/util/pool"
	"math/rand"
	"net"
)

const mClientPoolGroupSum = 10

type ClientPool struct {
	*log.Logger
	allSockets pool.MapPool // 所有连接
	linkSum    int32
}

// 初始化Clieng连接池
func (this *ClientPool) Init() {
	this.allSockets.Init(mClientPoolGroupSum)
}

func (this *ClientPool) SetLogger(l *log.Logger) {
	this.Logger = l
}

// 使用TCP连接新建一个Client
func (this *ClientPool) NewTCPClient(conn net.Conn,
	connHook ConnectHook) (*Client, error) {
	client := &Client{}
	client.SetLogger(this.Logger)
	client.InitTCP(conn, connHook)
	this.Add(client)
	return client, nil
}

// 加载或存储
func (this *ClientPool) LoadOrStore(k string,
	v *Client) (*Client, bool) {
	vi, isLoad := this.allSockets.LoadOrStore(k, v)
	res := vi.(*Client)
	return res, isLoad
}

// 根据连接的 TmpID 获取一个连接
func (this *ClientPool) Get(tempid string) *Client {
	if vi, ok := this.allSockets.Load(tempid); ok {
		return vi.(*Client)
	}
	return nil
}

// 获取当前连接数量
func (this *ClientPool) Len() uint32 {
	if this.linkSum < 0 {
		return 0
	}
	return uint32(this.linkSum)
}

// 删除一个连接
func (this *ClientPool) remove(tmpid string) {
	if _, ok := this.allSockets.Load(tmpid); !ok {
		return
	}
	// 删除连接
	this.allSockets.Delete(tmpid)
	this.linkSum--
}

// 增加一个连接
func (this *ClientPool) Add(client *Client) {
	tmpid := client.GetTempID()
	_, isLoad := this.allSockets.LoadOrStore(tmpid, client)
	if !isLoad {
		this.linkSum++
	} else {
		this.allSockets.Store(tmpid, client)
	}
}

// 遍历连接池中的所有连接
func (this *ClientPool) Range(
	callback func(*Client) bool) {
	this.allSockets.Range(func(tk, tv interface{}) bool {
		if tk == nil || tv == nil {
			return true
		}
		if !callback(tv.(*Client)) {
			return false
		}
		return true
	})
}

func (this *ClientPool) Remove(tempid string) {
	if vi, ok := this.allSockets.Load(tempid); ok {
		client := vi.(*Client)
		// 关闭消息发送协程
		client.Shutdown()
		// 删除连接
		this.remove(tempid)
		client.Debug("[ClientPool.Remove] 断开连接 TmpID[%s] 当前连接数量"+
			" Len[%d]",
			tempid, this.Len())
		return
	}
}

// 随机获取指定类型的一个连接
func (this *ClientPool) GetRandom() *Client {
	tasklist := make([]string, 0)
	this.Range(func(client *Client) bool {
		tasklist = append(tasklist, client.GetTempID())
		return true
	})

	length := len(tasklist)
	if length > 0 {
		return this.Get(tasklist[rand.Intn(length)])
	}
	return nil
}
