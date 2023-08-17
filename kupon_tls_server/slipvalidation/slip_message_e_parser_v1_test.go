package slipvalidation

import (
	"encoding/hex"
	"fmt"
	goqr "github.com/nishant8887/go-qrcode"
	"github.com/skip2/go-qrcode"
	"math"
	"testing"
)

func boolsToBytes(t [][]bool) []byte {
	//k := math.Ceil(float64(len(t)+7) / 8)
	//sizeB := math.Ceil(k * k + 3)
	fmt.Println(len(t))
	k := len(t)
	k = int(math.Ceil(float64(k*k) / 8))

	b := make([]byte, k) //len(t)+7)/8

	j := 0
	for _, k := range t {
		for i, x := range k {
			if x {
				b[j] |= 1 << uint(i%8)
			}
			if (i != 0) && (i%8 == 0) {
				j++
			}
		}
		j++
	}
	fmt.Println(hex.Dump(b))
	return b
}

//10101010   10101010  1010
//10101010   10101010  1010
//10101010   10101010  1010

func TestQrcodeGeneration(t *testing.T) {
	code, err := goqr.New("http://www.kasat.al/k1/a8ecc093-d459-40c9-8bc0-04d57fb2adb8", goqr.L)
	if err != nil {
		t.Error(err)
	}
	matr := code.Matrix()
	bytes := boolsToBytes(matr)

	qq, err := qrcode.New("http://www.kasat.al/k1/a8ecc093-d459-40c9-8bc0-04d57fb2adb8", qrcode.Low)
	if err != nil {
		t.Error(err)
	}
	qq.DisableBorder = true

	err = qq.WriteFile(200, "qr.png")
	bits := qq.Bitmap()
	bytes = boolsToBytes(bits)
	pngBytes, err := qrcode.Encode("http://www.kasat.al/k1/a8ecc093-d459-40c9-8bc0-04d57fb2adb8", qrcode.Low, 200)
	fmt.Println(bits)
	fmt.Println(bytes)
	fmt.Println(pngBytes)
}
