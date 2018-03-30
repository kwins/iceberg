package frame

import (
	"bytes"
	"encoding/binary"
	log "github.com/kwins/iceberg/frame/icelog"
	"github.com/kwins/iceberg/frame/protocol"
	"io"
	"net"
	"time"
)

// EachReadBufSize buf大小
const EachReadBufSize = 1024 * 2048

// ProcessInComingPackFunc 处理网络中接收到的请求的回调函数定义
type ProcessInComingPackFunc func([]byte)

// RecvPack 用于从客户端连接中读取数据，不提供断线重连功能
func RecvPack(conn net.Conn) ([]byte, error) {
	recvedBuf := bytes.NewBuffer(nil)
	var length uint32
	var buf [EachReadBufSize]byte
	for {
		n, err := conn.Read(buf[0:])
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				log.Error("Read timeout")
			} else if err == io.EOF {
				// 连接断开
				log.Error("Connection break!")
			} else {
				log.Error("Read from connection failed!, detail:", err.Error())
			}

			conn.Close()
			return nil, err
		}

		recvedBuf.Write(buf[0:n])
		recvBytes := recvedBuf.Len()
		if recvBytes < protocol.HeaderLength {
			// 从TCP流中读取数据太少继续读
			log.Info("Keep recv...")
			continue
		}

		// 读包头的表示长度的字节
		leaderNumBuf := bytes.NewBuffer(recvedBuf.Bytes()[:protocol.HeaderLength])
		binary.Read(leaderNumBuf, binary.BigEndian, &length)
		if recvBytes < int(length) {
			// 从TCP流中读取数据还是太少,继续读
			log.Warn("Pack head shows size=%d, buf just recv %d bytes, keep receive.", length, recvBytes)
			continue
		}

		if recvBytes > int(length) {
			log.Error("Recv data much than a pack. this issue is unnormal in iceberg!!")
		}

		// 读到了完整的包,暂停读数据
		break
	}
	packbuf := recvedBuf.Next(int(length))
	return packbuf, nil
}

// ContinuousRecvPack 用于从长连接中持续读取数据
// 全双工的方式读取数据
func ContinuousRecvPack(conn net.Conn, cstmFunc ProcessInComingPackFunc) {
	recvedBuf := bytes.NewBuffer(nil)
	var buf [EachReadBufSize]byte
	var length uint32
	var headByte = make([]byte, 4)
	var tempDelay = 5 * time.Millisecond
	for {
		n, err := conn.Read(buf[0:])
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				time.Sleep(tempDelay)
				tempDelay *= 2
				if tempDelay > time.Second {
					tempDelay = time.Second
				}
				continue
			}
			log.Warnf("Read from connection failed!, detail=%s", err.Error())
			go cstmFunc(nil) // notice handler connection is broken.
			return
		}

		recvedBuf.Write(buf[0:n])
		if recvedBuf.Len() < protocol.HeaderLength {
			// 从TCP流中读取数据太少继续读
			log.Info("Keep recv...")
			continue
		}

		// 读到了完整的包, 消费掉接收缓存中的所有数据
		for recvedBuf.Len() >= protocol.HeaderLength {
			// 检查包头的4byte
			headByte = recvedBuf.Bytes()[:protocol.HeaderLength]
			length = uint32(headByte[3]) | uint32(headByte[2])<<8 | uint32(headByte[1])<<16 | uint32(headByte[0])<<24
			if recvedBuf.Len() < int(length) {
				// 从TCP流中读取数据还是太少,继续读
				log.Debugf("Pack head shows size=%d, buf just recv %d bytes, keep receive.", length, recvedBuf.Len())
				break
			}
			pack := make([]byte, length)
			copy(pack, recvedBuf.Next(int(length)))
			go cstmFunc(pack)
		}
	} // for conn.Read loop
}

// SendAll 往连接上发送数据
func SendAll(conn net.Conn, buf []byte) int {
	var sentbytes int
	var tempDelay = 5 * time.Millisecond
	for {
		n, err := conn.Write(buf[sentbytes:])
		sentbytes += n
		if err != nil {
			log.Errorf("Failed send data to %s detail=%s", conn.RemoteAddr().String(), err)
			if err == io.EOF {
				return sentbytes
			}

			time.Sleep(tempDelay)
			if tempDelay < time.Second {
				tempDelay *= 2
			}
			continue
		}
		if sentbytes == len(buf) {
			return sentbytes
		}
	}
}
