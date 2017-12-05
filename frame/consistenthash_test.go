package frame

var chash = NewConsistentHash()

// func TestAddNode(t *testing.T) {
// 	// node为空的情况下
// 	if _, ok := chash.Locate([]byte("hello")); ok {
// 		t.Error("The ring is empty, should return false")
// 	}

// 	if !chash.AddNode([]byte("hello"), "localhost:3918", make([]byte, 0)) {
// 		t.Error("AddNode failed")
// 	}
// 	if len(chash.ring) != len(chash.nodeList) || len(chash.ring) != 1 {
// 		t.Error("node count should == 1, but len(chash.ring) ==", len(chash.ring), " len(chash.nodeList)==", len(chash.nodeList))
// 	}

// 	// 在只有一个node的情况下
// 	if _, ok := chash.Locate([]byte("hell")); !ok {
// 		t.Error("AddNode failed! can't locate node")
// 	}

// 	if !chash.AddNode([]byte("wd"), "localhost:3919", make([]byte, 0)) {
// 		t.Error("AddNode failed")
// 	}
// 	if len(chash.ring) != len(chash.nodeList) || len(chash.ring) != 2 {
// 		t.Error("node count should == 0, but len(chash.ring) ==", len(chash.ring), " len(chash.nodeList)==", len(chash.nodeList))
// 	}
// }

// func TestRmNode(t *testing.T) {
// 	chash.RmNode([]byte("hello"))
// 	if len(chash.ring) != len(chash.nodeList) || len(chash.ring) != 1 {
// 		t.Error("node count should == 1, but len(chash.ring) ==", len(chash.ring), " len(chash.nodeList)==", len(chash.nodeList))
// 	}
// 	v := _hash([]byte("hello"))
// 	t.Log(v)
// 	if _, found := chash.ring[v]; found {
// 		t.Error("the key[hello] should be removed!")
// 	}

// 	chash.RmNode([]byte("wd"))
// 	if len(chash.ring) != len(chash.nodeList) || len(chash.ring) != 0 {
// 		t.Error("node count should == 0, but len(chash.ring) ==", len(chash.ring), " len(chash.nodeList)==", len(chash.nodeList))
// 	}

// 	if _, ok := chash.Locate([]byte("hello")); ok {
// 		t.Error("The ring is empty, should return false, but", ok)
// 	}
// }

// func TestClear(t *testing.T) {
// 	if !chash.AddNode([]byte("wd"), "172.16.7.220:37126", make([]byte, 0)) {
// 		t.Error("AddNode failed")
// 	}

// 	chash.Clear()

// 	if len(chash.ring) != len(chash.nodeList) || len(chash.ring) != 0 {
// 		t.Error("node count should == 0, but len(chash.ring) ==", len(chash.ring), " len(chash.nodeList)==", len(chash.nodeList))
// 	}
// }

// func TestLocatePrecise(t *testing.T) {
// 	if !chash.AddNode([]byte("172.16.7.220:3163"), "172.16.7.220:3163", make([]byte, 0)) {
// 		t.Error("AddNode failed")
// 	}

// 	if !chash.AddNode([]byte("172.16.7.220:37126"), "172.16.7.220:37126", make([]byte, 0)) {
// 		t.Error("AddNode failed")
// 	}

// 	if !chash.AddNode([]byte("172.16.7.220:37136"), "172.16.7.220:37136", make([]byte, 0)) {
// 		t.Error("AddNode failed")
// 	}

// 	if _, ok := chash.LocatePrecise([]byte("172.16.7.220:37126")); !ok {
// 		t.Error("The ring have value , should return true, but", ok)
// 	} else {
// 		t.Log("success Locate:172.16.7.220:37126.")
// 	}

// 	if _, ok := chash.LocatePrecise([]byte("172.16.7.220:3163")); !ok {
// 		t.Error("The ring have value , should return true, but", ok)
// 	} else {
// 		t.Log("success Locate:172.16.7.220:3163.")
// 	}
// }
// func TestAlarm(t *testing.T) {
// 	AlarmEvent("test", "wangzhenkui@laoyuegou.com", "支付通知失败", "aaaaaaaaaaaa")
// }
