package frame

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"math"
	"sort"
	"sync"

	log "github.com/kwins/iceberg/frame/icelog"
)

/*
内部结构，保持节点有序的二叉堆
*/
type _NodeListSeq []uint32

func (h *_NodeListSeq) Len() int           { return len(*h) }
func (h *_NodeListSeq) Less(i, j int) bool { return (*h)[i] < (*h)[j] }
func (h *_NodeListSeq) Swap(i, j int)      { (*h)[i], (*h)[j] = (*h)[j], (*h)[i] }

func (h *_NodeListSeq) Insert(x interface{}) {
	// NOTE 可优化点，不直接append, 找到新节点的位置对后面的节点做移位后再插入
	// 这样就不用每次插入都做一次排序了
	*h = append(*h, x.(uint32))
	sort.Sort(h)
	log.Debug("insert node successful.")
}

func (h *_NodeListSeq) Remove(x interface{}) bool {
	if h.Len() == 0 {
		return false
	}

	i := sort.Search(h.Len(), func(i int) bool { return (*h)[i] >= x.(uint32) })
	if i < h.Len() && (*h)[i] == x.(uint32) {
		log.Debugf("remove node from nodeList:%d len:%d", i, h.Len())
		*h = append((*h)[:i], (*h)[i+1:]...)
		return true
	} else {
		return false
	}
}

// Node 实例节点
// @remoteAddr 节点监听地址
// @reqNo 节点当前负载状况
type Node struct {
	remoteAddr string
	reqNo      int64
}

/*
NewConsistentHash 创建并初始化一个新的一致性哈希实例
*/
func NewConsistentHash() *ConsistentHash {
	ch := new(ConsistentHash)
	ch.ring = make(map[uint32]*Node)
	// ch.virtualNodes = make(map[uint32][]uint32)
	// ch.virtual2realNode = make(map[uint32]uint32)
	return ch
}

/*
ConsistentHash 一致性哈希类

该类维护哈希环并提供hash接口
我们限制hash的值空间在uint32的表示范围内
*/
type ConsistentHash struct {
	ring     map[uint32]*Node // 节点到远端地址的字典
	nodeList _NodeListSeq     // ring当中key的有序列表
	sync.RWMutex
	// virtualNodes     map[uint32][]uint32 // 真实节点拥有的虚拟节点
	// virtual2realNode map[uint32]uint32   // 虚拟节点指向的真实节点
}

// Leastload 返回服务实例中负载最小的节点
func (chash *ConsistentHash) Leastload() string {
	if len(chash.ring) == 0 {
		log.Warn("connsistent hash circle is nil")
		return ""
	}

	var no uint32
	var remodeAddr string
	var minmum = int64(math.MaxInt64)
	chash.RLock()
	for k, node := range chash.ring {
		if node.reqNo < minmum {
			minmum = node.reqNo
			remodeAddr = node.remoteAddr
			no = k
		}
	}
	chash.RUnlock()
	if remodeAddr == "" {
		chash.ring[0].reqNo++
		return chash.ring[0].remoteAddr
	}
	chash.ring[no].reqNo++
	return remodeAddr
}

// Find find node
func (chash *ConsistentHash) Find(key []byte) *Node {
	if len(chash.nodeList) == 0 {
		log.Warn("The ring is empty!")
		return nil
	}
	return chash.find(key)
}

/*
Locate 根据hash key返回对应的后台服务的地址
*/
func (chash *ConsistentHash) Locate(key []byte) (string, bool) {
	if len(chash.nodeList) == 0 {
		return "", false
	}
	if n := chash.find(key); n != nil {
		return n.remoteAddr, true
	}
	return chash.ring[chash.nodeList[0]].remoteAddr, true
}

func (chash *ConsistentHash) find(key []byte) *Node {
	v := _hash(key)
	nodeLength := len(chash.nodeList)
	i := sort.Search(nodeLength, func(i int) bool { return chash.nodeList[i] >= v })
	if i < chash.nodeList.Len() && chash.nodeList[i] == v {
		return chash.ring[chash.nodeList[i]]
	}
	return nil
}

/*
AddNode 增加一个节点

key 要增加的节点key
svrAddr 新节点的监听地址
realNodeKey 如果长度不为0说明要增加的是一个虚拟节点。realNodeKey里保存就是虚拟节点对应的真实节点的key
*/
func (chash *ConsistentHash) AddNode(svrAddr string) bool {
	hashed := _hash([]byte(svrAddr))

	if v, found := chash.ring[hashed]; found {
		log.Warnf("Hash crash, chash node [%s:%d] is existed in ring [%s:%d]", svrAddr, hashed, v.remoteAddr, hashed)
		return false
	}

	node := new(Node)
	node.remoteAddr = svrAddr
	chash.Lock()
	chash.ring[hashed] = node
	chash.Unlock()
	chash.nodeList.Insert(hashed)
	log.Debugf("Add a new node %s into hash ring,hashed=%d", svrAddr, hashed)

	return true
}

/*
RmNode 删除一个节点，如果该节点有虚拟节点将一并清除
返回值  string 被删除的节点的远端地址
*/
func (chash *ConsistentHash) RmNode(key []byte) string {

	v := _hash(key)
	l, found := chash.ring[v]
	if !found {
		log.Warn("Can't remove node, because the node is not exist.")
		if chash.nodeList.Len() > 0 {
			return chash.ring[chash.nodeList[0]].remoteAddr
		} else {
			return ""
		}
	}

	// 在有序的节点key中找出该节点key的下标
	if !chash.nodeList.Remove(v) {
		log.Warn("The node is not exist in nodelist, but exist in ring, Data is not consistent!!!")
	}
	delete(chash.ring, v) // 从ring中删除节点

	// if _, found := chash.virtualNodes[v]; found {
	// 	// 该节点含有虚拟节点
	// 	// 清除相关的虚拟节点
	// 	delete(chash.virtualNodes, v)
	// }

	// if realv, found := chash.virtual2realNode[v]; found {
	// 	// 该节点是一个虚拟节点
	// 	// 删除该节点指向的真实节点中关于该虚拟节点的纪录
	// 	if vl, found := chash.virtualNodes[realv]; found {
	// 		for i := 0; i < len(vl); i++ {
	// 			virtualV := vl[i]
	// 			if virtualV == v {
	// 				chash.virtualNodes[realv] = append(vl[:i], vl[i+1:]...)
	// 				i--
	// 			}
	// 		}
	// 	}
	// }

	return l.remoteAddr
}

/*
Clear 清除所有节点
*/
func (chash *ConsistentHash) Clear() {
	// log.Info("Clear nodeList.")

	chash.ring = make(map[uint32]*Node)
	chash.nodeList = _NodeListSeq{}
	// chash.virtualNodes = make(map[uint32][]uint32)
	// chash.virtual2realNode = make(map[uint32]uint32)
}

/*
AllNode 返回所有节点地址
*/
func (chash *ConsistentHash) AllNode() []string {
	var alladdr []string
	for _, v := range chash.ring {
		alladdr = append(alladdr, v.remoteAddr)
	}

	return alladdr
}

func _hash(key []byte) uint32 {
	md5Inst := md5.New()
	md5Inst.Write(key)
	result := md5Inst.Sum([]byte(""))

	var value uint32
	buf := bytes.NewBuffer(result)
	err := binary.Read(buf, binary.LittleEndian, &value)
	if err != nil {
		log.Error("Calculate hash failed!")
	}

	return value
}
