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

func TestNanoID(t *testing.T) {
	k, err := NewNanoID()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", k)
	k, err = NewNanoID(5)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", k)
	i := 0
	m := make(map[string]int)
	for {
		if i > 1000000 {
			break
		}
		i++
		k, err = NewNanoID(10)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%v %v", k, i)
		m[k]++
	}
	for k, cont := range m {
		if cont > 1 {
			t.Logf("re:%v c:%v", k, cont)
		}
	}
}
