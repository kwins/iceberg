package icelog

import "testing"

func TestConsoleLog(t *testing.T) {
	Debug("a=1", " b=2")
	Debugf("a=%d b=%d", 1, 2)
	Info("a=1", " b=2")
	Infof("a=%d b=%d", 1, 2)
	Warn("a=1", " b=2")
	Warnf("a=%d b=%d", 1, 2)

	Error("a=1", " b=2")
	Errorf("a=%d b=%d", 1, 2)

	Fatal("a=1", " b=2")
	Fatalf("a=%d b=%d", 1, 2)
}

// func TestArgs(t *testing.T) {
// 	t.Log(os.Args[0])
// }

// func TestFileWriter(t *testing.T) {
// 	fw := NewFileWriter("/Users/quinn/goproj/laoyuegou.com/src/iceberg/frame/icelog/ice.log")
// 	for i := 0; i < math.MaxInt64; i++ {
// 		fw.Write(fmt.Sprint(i) + "\n")
// 		time.Sleep(time.Microsecond * 10)
// 	}
// 	fw.Close()
// }

// func TestFileWriter(t *testing.T) {
// 	SetLog("/Users/quinn/goproj/laoyuegou.com/src/iceberg/frame/icelog/ice.log", "debug")
// 	Debug("c==1", " d=2")
// 	Debugf("c==%d d=%d", 1, 2)
// 	Info("c==1", " d=2")
// 	Infof("c==%d d=%d", 1, 2)
// 	Warn("c==1", " d=2")
// 	Warnf("c==%d d=%d", 1, 2)

// 	Error("c==1", " d=2")
// 	Errorf("c==%d d=%d", 1, 2)

// 	Fatal("c==1", " d=2")
// 	Fatalf("c==%d d=%d", 1, 2)
// 	Close()
// }

// func TestFileWalk(t *testing.T) {
// 	filepath.Walk("/Users/quinn/goproj/laoyuegou.com/src/iceberg/frame/icelog/log.go", func(path string, info os.FileInfo, err error) error {
// 		if info == nil {
// 			return err
// 		}
// 		if info.IsDir() {
// 			return nil
// 		}
// 		t.Log(info.Name())
// 		return nil
// 	})
// }
