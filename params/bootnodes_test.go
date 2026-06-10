package params

import "testing"

const silaMainnetBootnode = "enode://2b91077d5fff048d13899c93d7f6d1391e182cd24b9a877355493328b9519aa59c5f7d90fa23fd315978e5129f94fe9ec99ffaf913acf19b57855cfd98f07ce0@192.248.181.185:30303"

func TestSilaMainnetBootnodes(t *testing.T) {
	if len(SilaMainnetBootnodes) == 0 {
		t.Fatal("SilaMainnetBootnodes is empty")
	}
	if SilaMainnetBootnodes[0] != silaMainnetBootnode {
		t.Fatalf("unexpected Sila mainnet bootnode: got %q want %q", SilaMainnetBootnodes[0], silaMainnetBootnode)
	}
}
