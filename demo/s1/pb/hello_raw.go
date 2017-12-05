package hello

// Get 实现Raw接口
func (r *HelloRequest) Get() []byte {
	return []byte(r.GetName())
}

// Set 实现Raw接口
func (r *HelloRequest) Set(b []byte) error {
	r.Name = string(b)
	return nil
}

// Get 实现Raw接口
func (r *HelloResponse) Get() []byte {
	return []byte(r.GetMessage())
}

// Set 实现Raw接口
func (r *HelloResponse) Set(b []byte) error {
	r.Message = string(b)
	return nil
}
