# bufio包解析

当前版本`go1.15`
```shell
tperam@Home-Ubuntu:~$ go version
go version go1.15 linux/amd64
```

一般来说，缓冲流io，是用来减少io操作的。假设我们每次读取量非常小，但是有序读取非常多次。在这种时候，缓冲流io将会派上大用场，他可以减少大量的读操作

那我们来思考一下，缓存流io是怎么实现减少io次数的？

一般来说，缓冲流io的实现方式是这样的

我们以读为例：

- 一次从io中读取大量数据。
- 我们每次进行读取数据时从缓存中读取
- 当缓存中的数据被读完，我们重新获取一次大量数据。直到读取到文件结束。

按照以上理论，如果我们的读取量不在一个特定的范围呢，可能会导致更大的问题。

- 当缓冲流过大，每次读取量过小 -> 占用过多的内存，并且是无效的
- 当缓冲流过小，每次的读取量过大 -> 并没有达到一个缓冲的效果。可能比原先io操作次数还多

基于以上理论，我们已经可以自己实现一个简单的读缓冲流。但go官方包中已经提供，我们没有什么必要的需求对其进行重写。所以我们在这里进行一点简单学习

我们当前项目使用了`br.ReadLine()`并且引发了一些小问题。所以我们展开其源码。找出问题，避免下次出错。

```go
func (b *Reader) ReadLine() (line []byte, isPrefix bool, err error) {
	line, err = b.ReadSlice('\n')
	if err == ErrBufferFull {
		// Handle the case where "\r\n" straddles the buffer.
		if len(line) > 0 && line[len(line)-1] == '\r' {
			// Put the '\r' back on buf and drop it from line.
			// Let the next call to ReadLine check for "\r\n".
			if b.r == 0 {
				// should be unreachable
				panic("bufio: tried to rewind past start of buffer")
			}
			b.r--
			line = line[:len(line)-1]
		}
		return line, true, nil
	}

	if len(line) == 0 {
		if err != nil {
			line = nil
		}
		return
	}
	err = nil

	if line[len(line)-1] == '\n' {
		drop := 1
		if len(line) > 1 && line[len(line)-2] == '\r' {
			drop = 2
		}
		line = line[:len(line)-drop]
	}
	return
}
```

`br.ReadLine()`当前方法返回三个值 `line []byte`,`isPrefix bool`,`err error`

`line`代表读取的值，`isPrefix`代表当前是否为前半段，`err`代表错误

当前方法关键点在于 `line, err = b.ReadSlice('\n')`
这行代码用于读取`line`，剩下代码是对与读取数据的一些操作。

`if`操作所执行的内容如下：

1. 如果当前行没读完，则会抛出错误`ErrBufferFull`，判断错误最后一个字符是否时`\r`，如果是则删除并且返回`false`。否则告诉用户`isPrefix=true`
2. 当前没有数据则返回`nil`
3. 判断倒数第一个`byte`是否为`\n` 如果是则`drop=1`，继续判断倒数第二个`byte`是否等于`\r`（windows下的换行符为`\r\n`）如果是则`drop=2`。将其切割，

基于以上代码。`ReadLine()`只是`b.ReadSlice()`的一个包装方法。所以我们继续解析`ReadSlice()`

```go
func (b *Reader) ReadSlice(delim byte) (line []byte, err error) {
	s := 0 // search start index
	for {
		// Search buffer. 
		if i := bytes.IndexByte(b.buf[b.r+s:b.w], delim); i >= 0 {
			i += s 
			line = b.buf[b.r : b.r+i+1]
			b.r += i + 1
			break
		}

		// Pending error?
		if b.err != nil {
			line = b.buf[b.r:b.w]
			b.r = b.w
			err = b.readErr()
			break
		}

		// Buffer full? 
		if b.Buffered() >= len(b.buf) {
			b.r = b.w
			line = b.buf
			err = ErrBufferFull
			break
		}

		s = b.w - b.r // do not rescan area we scanned before

		b.fill() // buffer is not full
	}

	// Handle last byte, if any.
	if i := len(line) - 1; i >= 0 {
		b.lastByte = int(line[i])
		b.lastRuneSize = -1
	}

	return
}
```

先对第一个if进行解析

```go
// Search buffer. 搜索缓存
// 通过阅读当前代码，我们大概能得知 b.buf 这个切片是我们的缓存。
// i 就为获取在指定的缓存切片中 查找 delim符号，并返回下标
if i := bytes.IndexByte(b.buf[b.r+s:b.w], delim); i >= 0 {
    // s 为开始搜索的下标 i+s = 当前符号的位置
    i += s 
    // b.r = 当前读取的位置
    line = b.buf[b.r : b.r+i+1]
    // 防止重复检索(防止再次读取数据)
    b.r += i + 1
    break
}
```

如果当前缓冲区包含`delim`则将其截断并将数据返回。

当前我们已经了解了大概的结构。当前我们需要了解，`b.buf`与`b.r`以及`b.w`这三个值是怎么生成的。

我们回到最开始的地方。也就是

```go
br := bufio.NewReader(file)
```

我们查看一下`NewReader`加载了什么，他初始化了哪些内容

```go
func NewReader(rd io.Reader) *Reader {
	return NewReaderSize(rd, defaultBufSize)
}
```

当前方法调用了`NewReaderSize`，传入了一个`defaultBufSize`

我们找到`defaultBufSize`的定义

```go
const (
	defaultBufSize = 4096
)
```

我们再继续深入，查看`NewReaderSize()`方法

```go
// NewReaderSize returns a new Reader whose buffer has at least the specified
// size. If the argument io.Reader is already a Reader with large enough
// size, it returns the underlying Reader.
func NewReaderSize(rd io.Reader, size int) *Reader {
	// Is it already a Reader?
    // 将io.Reader的实现类强行转换成 *Reader
	b, ok := rd.(*Reader)
    // 判断转换是否成功 并且b.buf 是否大于等于默认缓冲数 （ 4096
	if ok && len(b.buf) >= size {
        // 如果是，则意味着当前我们传入的是缓冲流。并且他是符合条件的。我们直接将其返回
		return b
	}
    // 找到 minReadBufferSize的定义
    // const minReadBufferSize = 16
    // 如果缓冲小于最小值，则将其提高到16
	if size < minReadBufferSize {
		size = minReadBufferSize
	}
    // 创建一个新的缓冲流
	r := new(Reader)
	r.reset(make([]byte, size), rd)
	return r
}
```

对于`r.reset(make([]byte, size), rd)`进行解析

```go
func (b *Reader) reset(buf []byte, r io.Reader) {
	*b = Reader{
		buf:          buf,
		rd:           r,
		lastByte:     -1,
		lastRuneSize: -1,
	}
}
```

我们在展开 `bufio.Reader`结构体，将我们已知的变量标记出来

```go
type Reader struct {
	buf          []byte
	rd           io.Reader // reader provided by the client
	r, w         int       // buf read and write positions
	err          error
	lastByte     int // last byte read for UnreadByte; -1 means invalid
	lastRuneSize int // size of last rune read for UnreadRune; -1 means invalid
}
```

- `buf` 缓冲字节数组
- `rd` input流
- `r,w` 字节读和写位置

我们回到 `ReadSlice` 模拟一遍从第一次开始运行

```go
func (b *Reader) ReadSlice(delim byte) (line []byte, err error) {
	s := 0 // search start index
	for {
		// Search buffer. 
        // 当前不会执行，因为我们b.buf为空，无法找到任何数据
		if i := bytes.IndexByte(b.buf[b.r+s:b.w], delim); i >= 0 {
			i += s 
			line = b.buf[b.r : b.r+i+1]
			b.r += i + 1
			break
		}

		// Pending error?
        // b.err = nil
		if b.err != nil {
			line = b.buf[b.r:b.w]
			b.r = b.w
			err = b.readErr()
			break
		}

		// Buffer full? 
        // b.Buffered() 为 获取 b.w-b.r的值。当前我们两项为默认值 0 
		if b.Buffered() >= len(b.buf) {
			b.r = b.w
			line = b.buf
			err = ErrBufferFull
			break
		}
		// s = b.w-b.r = 0
		s = b.w - b.r // do not rescan area we scanned before
		// 填充方法。我们第一次循环只会执行当前方法。所以我们分析一下他里面干了什么
		b.fill() // buffer is not full
	}

	// Handle last byte, if any.
	if i := len(line) - 1; i >= 0 {
		b.lastByte = int(line[i])
		b.lastRuneSize = -1
	}

	return
}
```

由于我们第一次执行

所有`if`条件都不成立。所以我们只会执行 `b.fill()`。我们即将对其进行解析

```go
// fill reads a new chunk into the buffer.
// 填充新的大块到缓冲中
func (b *Reader) fill() {
	// Slide existing data to beginning.
    // 当前 b.r = 0 
	if b.r > 0 {
		copy(b.buf, b.buf[b.r:b.w])
		b.w -= b.r
		b.r = 0
	}
    // b.w = 0 ,len(buf) = 4096
	if b.w >= len(b.buf) {
		panic("bufio: tried to fill full buffer")
	}

	// Read new data: try a limited number of times.
    // 读取新数据，尝试限制数字
	for i := maxConsecutiveEmptyReads; i > 0; i-- {
        // 调用默认io进行读取
        // 当前 b.w = 0
        // 所以当前将会读取 4096个字符
        // 根据官方文档 n 为读取到的字符数
		n, err := b.rd.Read(b.buf[b.w:])
		if n < 0 {
			panic(errNegativeRead)
		}
        // b.w = 0 + 4096
		b.w += n
        // 出错，则记录错误并且返回
		if err != nil {
			b.err = err
			return
		}
        // 读取完成则直接返回
		if n > 0 {
			return
		}
	}
	b.err = io.ErrNoProgress
}
```

那么，当前我们的 `b.buf`就从文件中读取到了4096个字节

### **总结**

我们现在已经知道当前包的运行过程了

1. 获取一个`io`流
2. 判断他是否可以直接转换成 `bufio.Reader`
3. 不可以则生成一个`bufio.Reader`。默认缓冲大小为`4096`
4. 当我们调用`ReadLine()`方法时。他会从缓冲字节数组中进行读取
5. 当我们缓冲字节为空，或者已经当前缓冲区已经被检索过。我们将从文件中重新读取数据，存放进缓冲流
6. 返回 -> 4. 
7. 直到文件结束

我们通过代码完整的解释一边上述过程

首先先获取一个`bufio.Reader`结构体的实例化

```go
type Reader struct {
	buf          []byte
	rd           io.Reader // reader provided by the client
	r, w         int       // buf read and write positions
	err          error
	lastByte     int // last byte read for UnreadByte; -1 means invalid
	lastRuneSize int // size of last rune read for UnreadRune; -1 means invalid
}
```

也就是通过`bufio.NewReader()`

```go
const (
	defaultBufSize = 4096
)
func NewReader(rd io.Reader) *Reader {
	return NewReaderSize(rd, defaultBufSize)
}
```

展开`NewReaderSize`

```go
func NewReaderSize(rd io.Reader, size int) *Reader {
	// Is it already a Reader?
    // 判断传入io是否可以直接转换成 bufio.Reader
	b, ok := rd.(*Reader)
    // 如果可以，并且缓冲流大于当前预定缓冲流，则将其直接返回
	if ok && len(b.buf) >= size {
		return b
	}
    // 判断设置缓冲流是否小于最小缓冲流 也就是16个字节
	if size < minReadBufferSize {
        // 小于则将其置为16
		size = minReadBufferSize
	}
    // 实例化一个Reader
	r := new(Reader)
    // 传入预定缓冲长度的[]byte，以及原先io流
	r.reset(make([]byte, size), rd)
	return r
}
```

通过以上代码我们就获取到了一个 `bufio.Reader`结构体，暂时为空

当我们第一次调用 `br.ReadLine()`时

```go
func (b *Reader) ReadLine() (line []byte, isPrefix bool, err error) {
    // 调用 
	line, err = b.ReadSlice('\n')
    // 判断是否因为缓冲满了，当前行没读完整
	if err == ErrBufferFull {
		// Handle the case where "\r\n" straddles the buffer.
		if len(line) > 0 && line[len(line)-1] == '\r' {
			// Put the '\r' back on buf and drop it from line.
			// Let the next call to ReadLine check for "\r\n".
			if b.r == 0 {
				// should be unreachable
				panic("bufio: tried to rewind past start of buffer")
			}
			b.r--
			line = line[:len(line)-1]
		}
		return line, true, nil
	}
	// 判断长度是否 = 0 如果等于 0 代表着当前行没有任何数据
	if len(line) == 0 {
        // 判断是否出错，如果出错则将 line 置为 nil
		if err != nil {
			line = nil
		}
		return
	}
    // 经过以上判断 err 置为 nil 
	err = nil
	// 判断最后一个字符是否等于 `\n`
    // 如果等于则 drop = 1 代表我们需要将其截断
	if line[len(line)-1] == '\n' {
		drop := 1
        // 如果是windows 默认换行符为 \r\n 所以我们还需要判断一下 len(lin-2) 的置
		if len(line) > 1 && line[len(line)-2] == '\r' {
            // drop = 2 我们需要删除两个字符
			drop = 2
		}
        // 将\n截断
		line = line[:len(line)-drop]
	}
    // 将处理过后的数据返回
	return
}
```

我们展开 `b.ReadSlice()` 看看可能发生什么

```go
func (b *Reader) ReadSlice(delim byte) (line []byte, err error) {
	s := 0 // search start index 搜索开始下标
	for {
		// Search buffer.
        // 判断是否能在当前缓冲区中找到相应的分隔符
		if i := bytes.IndexByte(b.buf[b.r+s:b.w], delim); i >= 0 {
			i += s
			line = b.buf[b.r : b.r+i+1]
			b.r += i + 1
			break
		}

		// Pending error?
        // 判断是否出错，一般这个错误为 io.EOF
		if b.err != nil {
			line = b.buf[b.r:b.w]
			b.r = b.w
			err = b.readErr()
			break
		}

		// Buffer full?
        // 判断是否是 缓冲超出，如果是则进行一些调整
		if b.Buffered() >= len(b.buf) {
			b.r = b.w
			line = b.buf
			err = ErrBufferFull
			break
		}
		
		s = b.w - b.r // do not rescan area we scanned before

        // 第一个if 判断是否能在当前缓冲区中找到相应的分隔符
        // 第二个if 判断是否有错，比如 io.EOF
        // 第三个if 判断b.w-b.r 是否 > len(b.buf)
       	// 执行到这一步，代表上面的if全都不符合条件
		b.fill() // buffer is not full
	}

	// Handle last byte, if any.
	if i := len(line) - 1; i >= 0 {
		b.lastByte = int(line[i])
		b.lastRuneSize = -1
	}

	return
}

```

解析`b.fill`

```go
// fill reads a new chunk into the buffer.
func (b *Reader) fill() {
	// Slide existing data to beginning.
    // 判断b.r是否大于0，大于0代表是当前不是第一次运行
	if b.r > 0 {
        // 将剩余数据复制下来。
        // 当 b.buf[b.r:b.w] 不包含分隔符时，我们将 b.buf[b.r:b.w] 这一段字节数组置为下一段字节数组的开始
		copy(b.buf, b.buf[b.r:b.w])
		b.w -= b.r
        // b.r 从0开始读取
		b.r = 0
	}

	if b.w >= len(b.buf) {
		panic("bufio: tried to fill full buffer")
	}

	// Read new data: try a limited number of times.
    // 暂时不大知道有啥用，先跳过
	for i := maxConsecutiveEmptyReads; i > 0; i-- {
        // 如果我们是第二次读取数据，并且最后一段数据不是完整的。
        // 则这次读取内容长度为 4096 - b.w
		n, err := b.rd.Read(b.buf[b.w:])
		if n < 0 {
			panic(errNegativeRead)
		}
        // b.w 可能出现两种结果，
        // 1. 数据填满4096个字节，b.w = 4096
        // 2. 数据已经读完，无法填满4096。 那么b.w就不能设置为4096
        // 所以当前为 b.w+=n 而不是 b.w = len(b.buf)
		b.w += n
		if err != nil {
			b.err = err
			return
		}
		if n > 0 {
			return
		}
	}
	b.err = io.ErrNoProgress
}
```

我们已经分析完我们所需的代码。

### **结论**

1. 缓冲区默认为 4096，可以自行调用 `NewReadSize`进行初始化。
2. 如果我们单条数据长度为100，读第41条数据时。
3. 他会在`ReadSlice`中发现我们无法读完，剩余内容将暂时不会进行处理。将会调用 `b.fill()`去读取数据
4. `b.fill()`将会把当前前半段数据复制到下一次缓冲区搜索的头部。
5. `b.r`置为0，`b.w`置为读取到的数据 + 上一次剩余数据长度
6. 最后重新执行`ReadSlice`，判断是否可以读取完整。读取完整则直接返回。

当中还有些细节处理，比如当我们单条数据长度大于缓冲区总长度。我们暂时不提及。

