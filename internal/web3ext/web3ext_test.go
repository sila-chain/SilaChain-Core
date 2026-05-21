package web3ext

import "testing"

func TestSilaExecutionExtensionReplacesEthExport(t *testing.T) {
	if _, ok := Modules["eth"]; ok {
		t.Fatalf("eth execution extension should not be exported")
	}
	if _, ok := Modules["sila"]; !ok {
		t.Fatalf("sila execution extension should be exported")
	}
}
