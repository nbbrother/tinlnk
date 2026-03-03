package base62

const (
	alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	base     = uint64(62)
)

// Encode 将数字编码为Base62字符串
func Encode(num uint64) string {
	if num == 0 {
		return string(alphabet[0])
	}

	result := make([]byte, 0, 11) // Snowflake最大11位
	for num > 0 {
		result = append(result, alphabet[num%base])
		num /= base
	}

	reverse(result)
	return string(result)
}

// Decode 将Base62字符串解码为数字
func Decode(s string) uint64 {
	var num uint64
	for _, c := range []byte(s) {
		num = num*base + uint64(indexOf(c))
	}
	return num
}

func indexOf(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'A' && c <= 'Z':
		return int(c - 'A' + 10)
	case c >= 'a' && c <= 'z':
		return int(c - 'a' + 36)
	default:
		return 0
	}
}

func reverse(b []byte) {
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
}
