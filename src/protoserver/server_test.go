/*
unit testing for pack protoserver
*/
package protoserver

import "testing"

/*
logic testing for SayHello
*/
func TestSayHello(t *testing.T) {
	SayHello()
	if 1 == 3 {
		t.Errorf("SayHello failed")
	}
}

/*
performance testing for SayHello
*/
func BenchmarkSayHello(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SayHello()
	}
}
