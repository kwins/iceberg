package serve

import (
	"net/http"
	"strings"
	"sync"
)

// Router Gateway 前缀匹配算法Router
// path => handler
type Router struct {
	// services服务体系的入口
	srvHandler http.HandlerFunc

	// NotFound 入口
	notFound http.HandlerFunc
	rLocker  sync.RWMutex
	// 其他入口
	trees map[string]http.HandlerFunc
}

// NewRouter new router
func NewRouter(srvHanlder, notFoundHandler http.HandlerFunc) *Router {
	r := new(Router)
	r.srvHandler = srvHanlder
	r.notFound = notFoundHandler
	return r
}

// Add add path and handler to router
// path 必须是以 / 开始 并且 不能只有 /
func (r *Router) Add(path string, handler http.HandlerFunc) {
	if r.trees == nil {
		r.trees = make(map[string]http.HandlerFunc)
	}
	if path == "/" || len(path) < 2 {
		panic("bad path:" + path)
	}
	r.trees[path] = handler
}

// Hanlder get path's handler
func (r *Router) Hanlder(path string) http.HandlerFunc {
	if strings.HasPrefix(path, root) {
		return r.srvHandler
	}
	r.rLocker.RLock()
	if v, ok := r.trees[path]; ok {
		r.rLocker.RUnlock()
		return v
	}
	r.rLocker.RUnlock()
	return r.notFound
}
