package unique

import "testing"

func TestObfus(t *testing.T) {
	obfu, _ := New([]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA})
	c := obfu.Obfus(uint32(3493209676))
	p := obfu.Unobfus(c)
	t.Logf("Obfus:%+v p:%+v", c, p)
}

func TestUUID(t *testing.T) {
	t.Logf("%v", UUID())
}
