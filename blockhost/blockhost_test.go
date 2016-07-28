package blockhost

import (
	"testing"
	"os"
	"io/ioutil"
	"time"
)

func TestCreate(t *testing.T) {
	var tempDir string

	if td, err := ioutil.TempDir("", "blockhost"); err != nil {
		t.Fatal(err)
	} else {
		tempDir = td
	}
	defer os.RemoveAll(tempDir)

	bh, err := New(tempDir + "/bh.db", "bh", 24 * time.Hour)
	if err != nil {
		t.Error(err)
	}
	defer bh.Close()

}

func TestBlockDB(t *testing.T) {
	bh, err := New("/tmp/bh.db", "bh", 24 * time.Hour)
	if err != nil {
		t.Error(err)
	}
	defer bh.Close()

	err = bh.BlockActives()
	if err != nil {
		t.Error(err)
	}
}
