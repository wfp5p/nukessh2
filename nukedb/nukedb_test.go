package nukedb

import (
	"testing"
	"io/ioutil"
	"os"
	"time"
	"database/sql"
)

func TestCreate(t *testing.T) {

	someDate := time.Now()

	tempDir, e1 := ioutil.TempDir("", "nukedb")
	if e1 != nil {
		t.Fatal(e1)
	}
	defer os.RemoveAll(tempDir)

	n, err := New(tempDir + "/foo.db")

	if err != nil {
		t.Error("it failed")
	}

	n.Insert("128.143.12.12", someDate, 5)

	var r NukeRecord
	r, err = n.GetInfo("128.143.12.12")

	if err != nil || r.Expire != someDate.Unix() || r.Blocks != 5 {
		t.Errorf("GetInfo failed %v %v\n", err, r)
	}

	r, err = n.GetInfo("1.2.3.4")
	if err != sql.ErrNoRows {
		t.Errorf("GetInfo returned something when it shouldn't %v %v", err, r)
	}

	err = n.InsertExpire("128.143.12.12", someDate)
	if err != nil {
		t.Errorf("InsertExpire failed %v\n", err)
	}

	r, err = n.GetInfo("128.143.12.12")

	if err != nil || r.Expire != someDate.Unix() || r.Blocks != 6 {
		t.Errorf("GetInfo after update failed %v %v\n", err, r)
	}

	err = n.InsertExpire("128.143.5.7", someDate)
	if err != nil {
		t.Errorf("InsertExpire failed %v\n", err)
	}

	r, err = n.GetInfo("128.143.5.7")

	if err != nil || r.Expire != someDate.Unix() || r.Blocks != 1 {
		t.Errorf("GetInfo after update2 failed %v %v\n", err, r)
	}

	ips, e3 := n.GetExpires(someDate.AddDate(0,0,1))
	if e3 != nil || len(ips) != 2 {
		t.Errorf("GetExpires failed %v %v\n", ips, e3)
	}

	err = n.ClearExpire("128.143.5.7")

	r, err = n.GetInfo("128.143.5.7")
	if err != nil || r.Expire != 0 {
		t.Errorf("GetInfo after ClearExpire failed %v %v\n", err, r)
	}

	n.Purge(time.Now())
	_, err = n.GetInfo("128.143.5.7")
	if err != sql.ErrNoRows {
		t.Fatal("Purge failed to purge")
	}

	err = n.Close()
	if err != nil {
		t.Fatal(err)
	}
}

