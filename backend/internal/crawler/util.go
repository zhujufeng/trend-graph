// Package crawler - 公共辅助函数
package crawler

// urlEncode 简单 URL 编码：把中文等非 URL 安全字符编码为 %XX
//
// 不用 net/url.QueryEscape 是因为它会把空格变成 + ，某些接口对 + 解析不一致。
// 这里把空格保持为 %20，更通用。
func urlEncode(s string) string {
	if s == "" {
		return ""
	}
	const hexChars = "0123456789ABCDEF"
	var buf []byte
	// 走 UTF-8 字节，中文会被编码成 3 字节 %XX
	for i := 0; i < len(s); i++ {
		c := s[i]
		// 不需要编码的字符：字母、数字、几个安全符号
		if isURLSafeByte(c) {
			buf = append(buf, c)
			continue
		}
		if c == ' ' {
			buf = append(buf, '%', '2', '0')
			continue
		}
		buf = append(buf, '%', hexChars[c>>4], hexChars[c&15])
	}
	return string(buf)
}

// isURLSafeByte 判断一个字节是否 URL 安全字符
func isURLSafeByte(c byte) bool {
	if 'a' <= c && c <= 'z' {
		return true
	}
	if 'A' <= c && c <= 'Z' {
		return true
	}
	if '0' <= c && c <= '9' {
		return true
	}
	return c == '-' || c == '_' || c == '.' || c == '~'
}