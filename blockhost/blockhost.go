package blockhost

import (
	"fmt"
	"os"
	"nukessh/nukedb"
	"strings"
	"time"

	"github.com/gmccue/go-ipset"

	"log"
)

type BlockHost struct {
	nukeDB *nukedb.NukeDB
	ips *ipset.IPSet
	setname string
}

func New(dbname string, setname string) (*BlockHost, error) {

	if uid := os.Geteuid(); uid != 0 {
		return nil, fmt.Errorf("blockhost: not running with root privs")
	}

	var bh BlockHost
	bh.setname = setname

	if s, err := ipset.New(); err != nil {
		return nil, err
	} else {
		bh.ips = s
	}

	if err := bh.ips.Create(setname, "hash:ip"); err != nil {
		if strings.Contains(err.Error(), "set with the same name already exists") {
			bh.ips.Flush(setname)
		} else {
			return nil, fmt.Errorf("blockhost: error %s when trying to create ipset",err)
		}
	}

	if ndb, err := nukedb.New(dbname); err != nil {
		return nil, err
	} else {
		bh.nukeDB = ndb
	}

	return &bh, nil
}

func (bh BlockHost) Close() {
	// destroy the set
	_ = bh.ips.Destroy(bh.setname)

	// close the SQL connection
	_ = bh.nukeDB.Close()
}

// add all active blocks to the set
func (bh BlockHost) BlockDB() error {
	ips, err := bh.nukeDB.GetActive(time.Now())
	if err != nil {
		return err
	}

	for _, ip := range ips {
		if err := bh.ips.AddUnique(bh.setname, ip); err != nil {
			log.Fatal(err)
		}
	}
	return nil
}
