package bigint

import (
	"math/big"
)

var (
	Zero = big.NewInt(0)
	One  = big.NewInt(1)
)

// Clamp 对一个区间 [start, end] 内的值进行“约束”，它根据指定的size，计算从 start 开始的一个范围，如果范围小于或等于 size，返回 end；否则，返回 start + size - 1。
func Clamp(start, end *big.Int, size uint64) *big.Int {
	temp := new(big.Int)
	count := temp.Sub(end, start).Uint64() + 1
	if count <= size {
		return end
	}

	temp.Add(start, big.NewInt(int64(size-1)))
	return temp
}

func Matcher(num int64) func(*big.Int) bool {
	return func(bi *big.Int) bool { return bi.Int64() == num }
}

func WeiToETH(wei *big.Int) *big.Float {
	f := new(big.Float)
	f.SetString(wei.String())
	return f.Quo(f, big.NewFloat(1e18))
}

// StringToBigInt 将字符串转换为 *big.Int
func StringToBigInt(input string) (num *big.Int) {
	if input != "" {
		n, bol := big.NewInt(0).SetString(input, 10)
		if bol == true {
			num = n
		}
	}
	return
}
