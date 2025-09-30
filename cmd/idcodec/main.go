package main

import (
	"fmt"

	"github.com/bronystylecrazy/gx"
	"github.com/bronystylecrazy/gx/idcodec"
)

type User struct {
	ID   gx.ID
	Name string
}

func main() {
	// สร้าง codec แบบครบ config
	codec := idcodec.MustNewCodecFromSecret(idcodec.Config{
		Secret:  []byte("Sirawit Pratoomsuwan"),
		Version: 1,
		MacLen:  12,                             // ความยาว padding
		Domain:  []byte("crane-scoop-platform"), // domain separation
		Kind:    nil,                            // optional: default kind
	})

	// ตั้งเป็น default สำหรับ gx.ID
	gx.SetDefaultCodec(codec)

	// encode/decode ง่ายๆ
	s := codec.EncodeUint64(1234567890)
	fmt.Println("token:", s)
	id, _ := codec.DecodeToUint64(s)
	fmt.Println("decoded:", id)

	fmt.Println(gx.NewID(1234567890))
}
