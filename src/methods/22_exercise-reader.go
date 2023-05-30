package main

//import "golang.org/x/tour/reader"

// 实现一个 Reader 类型，它产生一个 ASCII 字符 'A' 的无限流。
// TODO: 给 MyReader 添加一个 Read([]byte) (int, error) 方法
type MyReader struct{}

func (mr MyReader) Read(bytes []byte) (int, error) {
	for i := range bytes {
		bytes[i] = 'A'
	}
	return len(bytes), nil
}

func main() {
	//reader.Validate(MyReader{})
}
